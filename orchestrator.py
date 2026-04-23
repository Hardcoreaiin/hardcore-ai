import os
import sys
import time
import json
import argparse
from typing import Dict, Any, Optional
import queue

# Ensure project directories are in path
sys.path.append(os.path.dirname(os.path.abspath(__file__)))

from hardware.gdb_adapter import GDBAdapter

class AntigravityOrchestrator:
    """
    Expert-grade Autonomous Monitor.
    Event-driven detection of crashes with zero hardware overhead.
    """

    def __init__(self, target_remote: str = "localhost:3333", elf_path: Optional[str] = None):
        self.gdb = GDBAdapter()
        self.target_remote = target_remote
        self.elf_path = elf_path
        self.evt_queue = queue.Queue()

    def run(self):
        print("🚀 HARDCOREAI: Launching Autonomous Intelligence Engine...")
        
        # Setup event handler
        self.gdb.on_notification = self._on_gdb_notification

        if not self.gdb.connect(target_remote=self.target_remote, elf_path=self.elf_path):
            print("❌ Error: Failed to established GDB link. Check OpenOCD.")
            sys.exit(1)

        print(f"📡 Passive Link Established: {self.target_remote}")
        print("🔍 Deep Monitoring Active. Waiting for crash event...")
        print("   (Antigravity Mode: 0% CPU Overhead on Target)")

        try:
            while True:
                # Wait for an event from the queue
                try:
                    evt_type, attrs = self.evt_queue.get(timeout=1.0)
                    
                    if evt_type == "stopped":
                        reason = attrs.get("reason", "unknown")
                        
                        # 1. HARD CRASH (Primary Target)
                        if reason == "signal-received":
                            sig = attrs.get("signal-name", "Unknown Signal")
                            if sig in ["SIGTRAP", "SIGINT"]:
                                # Likely GDB internal or trivial stop
                                print(f"ℹ️  Halt: {sig} (Standard Debug Stop)")
                                continue
                                
                            print(f"\n💥 [CRASH EVENT] Hardware Exception: {sig}")
                            self._capture_and_diagnose(attrs)
                            print("\n🔄 Resuming Passive Monitoring...")
                        
                        # 2. USER BREAKPOINT (Passive Observation)
                        elif reason == "breakpoint-hit":
                            print(f"📍 Breakpoint Hit at {attrs.get('frame', {}).get('addr', '???')}")
                            # We DON'T generate a crash report for breakpoints
                        
                        # 3. STEPPING (Ignore)
                        elif reason in ["end-stepping-range", "function-finished"]:
                            # Peaceful stepping, do nothing
                            pass
                        
                        else:
                            print(f"ℹ️  Target Halted: Reason={reason}")
                            
                except queue.Empty:
                    # Heartbeat
                    # sys.stdout.write(".")
                    # sys.stdout.flush()
                    continue

        except KeyboardInterrupt:
            print("\n👋 Monitoring terminated.")
        finally:
            self.gdb.disconnect()

    def _on_gdb_notification(self, evt_type: str, attrs: Dict[str, Any]):
        """ Dispatcher for GDB events. """
        self.evt_queue.put((evt_type, attrs))

    def _capture_and_diagnose(self, stop_attrs: Dict[str, Any]):
        """
        Deterministic capture of the triad (PC, LR, SP) + SCB registers.
        """
        print("📸 Performing zero-touch state capture...")
        
        try:
            # 1. Capture core architectural state
            regs = self.gdb.read_registers(["pc", "lr", "sp"])
            
            # 2. Capture bit-level fault registers
            scb = self.gdb.read_scb_fault_registers()
            
            # 3. Capture a chunk of stack for unwinding (optional but powerful)
            stack_raw = ""
            sp = regs.get("sp")
            if sp:
                stack_bytes = self.gdb.read_memory(sp, 128) # 128 bytes = 32 words
                stack_raw = stack_bytes.hex()

            payload = {
                "timestamp": time.time(),
                "reason": stop_attrs.get("reason"),
                "signal": stop_attrs.get("signal-name"),
                "pc": f"0x{regs.get('pc', 0):08X}",
                "lr": f"0x{regs.get('lr', 0):08X}",
                "sp": f"0x{regs.get('sp', 0):08X}",
                "stack_raw": stack_raw,
                "cfsr": f"0x{scb.get('cfsr', 0):08X}",
                "hfsr": f"0x{scb.get('hfsr', 0):08X}",
                "mmfar": f"0x{scb.get('mmfar', 0):08X}",
                "bfar": f"0x{scb.get('bfar', 0):08X}"
            }

            # Write to auto_fault.json (Continuous overwrite for extension watcher)
            with open("auto_fault.json", "w") as f:
                json.dump(payload, f, indent=4)
            
            print(f"✅ Context stored to auto_fault.json [PC: {payload['pc']}]")
            
        except Exception as e:
            print(f"❌ State capture failed: {e}")

def main():
    parser = argparse.ArgumentParser(description="HARDCOREAI Autonomous Monitor")
    parser.add_argument("--remote", default="localhost:3333")
    parser.add_argument("--elf", help="Symbol file")
    args = parser.parse_args()

    orchestrator = AntigravityOrchestrator(target_remote=args.remote, elf_path=args.elf)
    orchestrator.run()

if __name__ == "__main__":
    main()
