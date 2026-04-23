import time
import threading
import asyncio
from typing import Optional, Dict, Any, List
from hardware.openocd_adapter import OpenOCDAdapter # type: ignore
from hardware.gdb_adapter import GDBAdapter # type: ignore
from hardware.uart_adapter import UARTAdapter # type: ignore
from cli.diagnostic_engine import DiagnosticEngine # type: ignore
from firmware_engine.fault_decoder.decoder import HardFaultDecoder # type: ignore
import os
from dotenv import load_dotenv
from hardware.arch_selector import select_adapter # type: ignore
from learning_engine.ai_crash_analyzer import AICrashAnalyzer # type: ignore
import json

load_dotenv() # Load environments from .env file

class HardwareOrchestrator:
    """
    The brain of Phase 3. Coordinates OpenOCD, GDB, and UART.
    Polls target state and triggers analysis automatically.
    """

    def __init__(self, config: Dict[str, Any]):
        self.config = config
        
        # Use OpenOCD path from .env if it exists
        openocd_bin = os.getenv("OPENOCD_BIN", "openocd")
        self.openocd = OpenOCDAdapter(binary_path=openocd_bin)
        
        self.gdb = GDBAdapter()
        self.uart = UARTAdapter(config.get('uart_port', 'COM1'))
        
        # Internal engines
        rules_path = config.get('rules_path', os.path.join(os.path.dirname(__file__), "..", "rule_engine", "fault_rules.json"))
        self.decoder = HardFaultDecoder(rules_path)
        self.diag_engine = DiagnosticEngine()
        self.ai_analyzer = AICrashAnalyzer()
        
        self.arch = None # Will be detected on start
        self.running = False
        self.thread: Optional[threading.Thread] = None
        self.on_fault_detected = None # Callback for VS Code or CLI

    def start(self) -> bool:
        print("[Orchestrator] Starting Hardware Layer...")
        
        # 1. Start OpenOCD
        if not self.openocd.connect(self.config['interface'], self.config['target']):
            return False
            
        # 2. Start GDB
        if not self.gdb.connect(elf_path=self.config.get('elf_path')):
            self.openocd.disconnect()
            return False
            
        # 3. Start UART (Optional)
        self.uart.on_pattern_match = self._handle_uart_trigger
        self.uart.connect()
        
        self.running = True
        self.thread = threading.Thread(target=self._monitor_loop, daemon=True)
        if self.thread:
            self.thread.start()
            
        # 4. Detect Architecture
        self.arch = select_adapter(self.gdb)
        if self.arch:
            print(f"[Orchestrator] Detected architecture: {self.arch.__class__.__name__}")
        else:
            default_name = self.config.get('default_arch', 'CortexMAdapter')
            print(f"[Orchestrator] WARNING: Auto-detection failed. Falling back to {default_name}.")
            # Fallback to Cortex-M by default as it is the most common
            self.arch = CortexMAdapter()
            
        return True

    def stop(self):
        self.running = False
        self.uart.disconnect()
        self.gdb.disconnect()
        self.openocd.disconnect()

    def _monitor_loop(self):
        print("[Orchestrator] Monitoring target for faults...")
        while self.running:
            try:
                state = self.gdb.get_state()
                if state == "halted":
                    # Check if it halted due to a fault
                    self._perform_auto_analysis()
                    # Wait for user to resume or stop
                    while self.running and self.gdb.get_state() == "halted":
                        time.sleep(1)
                
                time.sleep(0.5)
            except Exception as e:
                print(f"[Orchestrator] Monitor error: {e}")
                break

    def _handle_uart_trigger(self, line: str):
        print(f"[Orchestrator] UART TRIGGER detected: {line}")
        # When UART detects a fault, we force a halt check
        self._perform_auto_analysis()

    def _run_ai_analysis(self, report: Dict[str, Any]):
        """Runs the slow AI analysis in a separate thread."""
        try:
            # 1. Get code snippet
            where = report.get("where", {})
            file_path = where.get("file")
            line_num = where.get("line")
            code_snippet = "N/A"
            
            if file_path and line_num and os.path.exists(file_path):
                code_snippet = self._get_code_snippet(file_path, line_num)
            
            # 2. Call AI
            loop = asyncio.new_event_loop()
            asyncio.set_event_loop(loop)
            ai_result = loop.run_until_complete(self.ai_analyzer.analyze(report, code_snippet))
            
            if ai_result:
                report["ai_analysis"] = ai_result
                # Update persistent file for VS Code
                self._save_report(report)
                # Trigger callback again with merged data
                if self.on_fault_detected and callable(self.on_fault_detected):
                    self.on_fault_detected(report)
                print("[Orchestrator] AI Analysis complete.")
        except Exception as e:
            print(f"[Orchestrator] AI background worker failed: {e}")

    def _save_report(self, report: Dict[str, Any]):
        """Persists the full report to a file for VS Code synchronization."""
        try:
            filename = "auto_fault.json"
            with open(filename, "w") as f:
                json.dump(report, f, indent=4)
            # print(f"[Orchestrator] Diagnostic report updated in {filename}")
        except Exception as e:
            print(f"[Orchestrator] Failed to save report: {e}")

    def _get_code_snippet(self, file_path: str, line_num: int, context: int = 5) -> str:
        """Extracts code snippet from file."""
        try:
            with open(file_path, "r", encoding="utf-8", errors="ignore") as f:
                lines = f.readlines()
                start = max(0, line_num - context - 1)
                end = min(len(lines), line_num + context)
                
                snippet = []
                for i in range(start, end):
                    prefix = "→ " if i == line_num - 1 else "  "
                    snippet.append(f"{prefix}{lines[i].rstrip()}")
                return "\n".join(snippet)
        except:
            return "Error reading code snippet."

    def _perform_auto_analysis(self):
        print("[Orchestrator] Capturing fault context...")
        
        try:
            # Check if arch is detected
            if not self.arch:
                self.arch = select_adapter(self.gdb)
            
            if not self.arch:
                print("[Orchestrator] WARNING: Analysis triggered but arch unknown. Falling back to Cortex-M.")
                self.arch = CortexMAdapter()

            # 1. Capture Architecture-Specific Registers
            raw_regs = self.arch.capture_registers(self.gdb)
            
            # 2. Extract common fields for legacy logging/UI
            pc = raw_regs.get("pc", 0)
            lr = raw_regs.get("lr", 0)
            
            # 3. Save to auto_fault.json for VS Code monitoring
            fault_payload = {
                "pc": hex(pc) if isinstance(pc, int) else pc,
                "lr": hex(lr) if isinstance(lr, int) else lr,
                "timestamp": time.time(),
                "arch": self.arch.__class__.__name__
            }
            # Add all other captured registers to payload
            for k, v in raw_regs.items():
                if k not in ["pc", "lr"]:
                    fault_payload[k] = hex(v) if isinstance(v, int) else v
            
            with open("auto_fault.json", "w") as f:
                import json
                json.dump(fault_payload, f, indent=4)
                print("[Orchestrator] Fault state saved to auto_fault.json")

            # 4. Perform Adapter-Specific Analysis
            raw_data = self.arch.analyze(raw_regs)
            
            # 5. Expert reasoning via Intelligence Layer
            final_report = self.diag_engine.analyze(raw_data)
            final_report["arch"] = self.arch.__class__.__name__
            final_report["raw_registers"] = raw_regs
            
            # Initial save for VS Code visibility (Awaiting AI)
            self._save_report(final_report)

            # 6. Trigger AI reasoning (Asynchronous/Non-blocking)
            threading.Thread(target=self._run_ai_analysis, args=(final_report,), daemon=True).start()

            if self.on_fault_detected and callable(self.on_fault_detected):
                self.on_fault_detected(final_report)
            else:
                print("\n" + "!"*40)
                print(" HARDCOREAI: LIVE FAULT DETECTED ")
                print("!"*40)
                print(f"TYPE: {final_report['fault_type']}")
                print(f"LOC : {final_report['where']['address']}")
                print("!"*40 + "\n")

        except Exception as e:
            print(f"[Orchestrator] Analysis failed: {e}")
