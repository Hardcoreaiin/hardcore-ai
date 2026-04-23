import json
import os
import sys
from typing import Dict, List, Any, Optional

# Add firmware_engine to path to allow import
sys.path.append(os.path.abspath(os.path.join(os.path.dirname(__file__), '..')))
from firmware_engine.memory_analyzer import MemoryAnalyzer
from firmware_engine.stack_unwinder import StackUnwinder

class DiagnosticEngine:
    """
    Expert reasoning engine for HARDCOREAI.
    Combines hardware register state, memory map analysis, and ELF symbols 
    to provide firmware-engineer-grade insights.
    """

    CONF_DETERMINISTIC = "Deterministic" # Proven by Hardware Bits
    CONF_HIGH          = "High"          # Strongly indicated by memory/PC
    CONF_MEDIUM        = "Medium"        # Heuristic match (stack/registers)
    CONF_LOW           = "Low"           # Generic assumption

    def __init__(self):
        pass

    def analyze(self, fault_data: Dict[str, Any], elf_info: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
        """
        Performs deep analysis of the fault context.
        """
        details = fault_data.get("details", [])
        addr_info = fault_data.get("address_info", {})
        pc_val = fault_data.get("crash_location", {}).get("address")
        evidence = []
        
        # 1. Memory Region Analysis
        mem_analysis = {"region": "Unknown", "valid": "Unknown"}
        if pc_val and isinstance(pc_val, str) and pc_val.startswith("0x"):
            try:
                addr_int = int(pc_val, 16)
                mem_analysis = MemoryAnalyzer.analyze_address(addr_int)
                evidence.append(f"PC resides in {mem_analysis['region']} ({mem_analysis.get('description', 'N/A')})")
            except ValueError:
                pass

        # 2. Timeline Reconstruction (FORCED Bit logic)
        timeline_notes = []
        if self._has_condition(details, "FORCED"):
            evidence.append("HFSR.FORCED=1 detected (Fault Escalation)")
            timeline_notes.append("A lower-priority fault (Bus/Usage/Mem) was escalated to HardFault due to priority or configuration.")

        # 3. Stack Analysis (Heuristic Unwinding)
        stack_trace = []
        stack_raw = fault_data.get("stack_raw")
        sp_val = fault_data.get("sp")
        if stack_raw and sp_val:
            try:
                sp_int = int(sp_val, 16)
                stack_bytes = bytes.fromhex(stack_raw)
                potential_calls = StackUnwinder.scan_stack(stack_bytes, sp_int)
                
                for call in potential_calls:
                    stack_trace.append({**call, "function": "Resolved via ELF" if elf_info else "Unknown"})
                
                if stack_trace:
                    evidence.append(f"Found {len(stack_trace)} potential return addresses on stack.")
            except:
                pass

        # 4. Base result structure
        result = {
            "fault_type": fault_data.get("fault_type", "Unknown Fault"),
            "confidence": self.CONF_LOW,
            "what_happened": "An unidentified exception occurred.",
            "evidence": evidence,
            "timeline": timeline_notes,
            "memory_analysis": mem_analysis,
            "stack_trace": stack_trace,
            "where": self._format_location(fault_data, elf_info),
            "likely_scenario": "Insufficient data to reconstruct the failure timeline.",
            "fix": ["Investigate the call stack.", "Verify hardware connections."],
            "raw_details": details
        }

        # 5. Expert Reasoning Engine
        self._reason_scenarios(result, details, addr_info, mem_analysis)

        return result

    def _reason_scenarios(self, result, details, addr_info, mem_analysis):
        """
        Expert-level diagnostic reasoning.
        """
        
        # 1. NULL Pointer Call (PC is 0x0)
        pc_val = result["where"]["address"]
        if pc_val == "0x00000000" or pc_val == "0x0":
            result["confidence"] = self.CONF_DETERMINISTIC
            if result["fault_type"] in ["Unknown Fault", "Halted", "Escalated to HardFault"]:
                result["fault_type"] = "Null Function Pointer Call"
            result["what_happened"] = "The CPU attempted to branch to address 0x0."
            result.get("evidence", []).append("PC is exactly 0x0 (Deterministic NULL branch)")
            result["likely_scenario"] = "A function pointer (callback) was used without being initialized. This is common in event-driven systems or driver callbacks."
            result["fix"] = [
                "Locate the function call leading to this crash using the LR (Link Register).",
                "Add 'if (callback != NULL)' before the call.",
                "Ensure all driver init functions are called before events start firing."
            ]
            return

        # 2. Undefined Instruction (UsageFault)
        if self._has_condition(details, "UNDEFINSTR"):
            result["confidence"] = self.CONF_DETERMINISTIC
            if result["fault_type"] in ["Unknown Fault", "Escalated to HardFault"]:
                result["fault_type"] = "Undefined Instruction"
            result.get("evidence", []).append("Detected Illegal/Undefined Instruction bit")
            
            if mem_analysis["region"] == "SRAM":
                result["confidence"] = self.CONF_HIGH
                result.get("evidence", []).append("PC is in SRAM during Instruction Fetch")
                result["likely_scenario"] = "Code execution jumped into RAM. Likely a 'stack smash' (buffer overflow overwriting return address)."
                result["fix"] = [
                    "Check for large local buffers (e.g., char buf[256]) without bounds checking.",
                    "Review recent 'memcpy' or 'sprintf' calls.",
                    "Check for stack overflow using the SP (Stack Pointer) captured."
                ]
            else:
                result["likely_scenario"] = "Jump to invalid code memory or corrupted flash image."
                result["fix"] = ["Verify firmware integrity via CRC.", "Check if MPU is blocking instruction prefetch."]

        # 3. Precise Data Access (BusFault)
        elif self._has_condition(details, "PRECISERR"):
            fault_addr = addr_info.get("BusFault Address", "Unknown")
            addr_int = int(fault_addr, 16) if isinstance(fault_addr, str) and fault_addr.startswith("0x") else 0
            
            result["confidence"] = self.CONF_DETERMINISTIC
            if result["fault_type"] in ["Unknown Fault", "Escalated to HardFault"]:
                result["fault_type"] = "Bus Access Error"
            result.get("evidence", []).append(f"Fault Address: {fault_addr}")
            
            if 0x40000000 <= addr_int <= 0x5FFFFFFF:
                result["likely_scenario"] = "Peripheral Access Fault. The peripheral clock (RCC) is likely DISABLED."
                result["fix"] = [
                    f"Ensure __HAL_RCC_..._CLK_ENABLE() is called for peripheral near {fault_addr}.",
                    "Check if the peripheral is in a low-power mode (Sleep/Stop).",
                    "Verify the peripheral address in the datasheet."
                ]
            elif addr_int < 0x1000:
                result["confidence"] = self.CONF_HIGH
                result.get("evidence", []).append("Fault address is in NULL pointer page (<4KB)")
                result["likely_scenario"] = "Null Pointer Dereference (Data Access)."
                result["fix"] = ["Check for NULL pointers in the line of code that crashed."]
            else:
                result["likely_scenario"] = "Access to invalid or protected memory address."
                result["fix"] = ["Check linker script boundaries.", "Verify pointer arithmetic."]

        # 4. Invalid State (INVSTATE / LSB check)
        elif self._has_condition(details, "INVSTATE"):
            result["confidence"] = self.CONF_DETERMINISTIC
            if result["fault_type"] in ["Unknown Fault", "Escalated to HardFault"]:
                result["fault_type"] = "Invalid State (Thumb/ARM Mismatch)"
            result.get("evidence", []).append("Invalid State bit set (e.g. INVSTATE)")
            result["likely_scenario"] = "Execution state mismatch. Likely a branch to an even address on Cortex-M (missed LSB=1)."
            result["fix"] = [
                "If using function pointers, ensure they are not cast incorrectly to even addresses.",
                "Verify assembly labels for function entry points have '.thumb_func' directive.",
                "Check for corrupted jump tables."
            ]

    def _has_condition(self, details: List[Dict[str, Any]], name: str) -> bool:
        return any(d.get("name") == name for d in details)

    def _format_location(self, fault_data: Dict[str, Any], elf_info: Optional[Dict[str, Any]]) -> Dict[str, Any]:
        """ Professional location formatting with ELF symbol priority. """
        loc = {
            "function": "Unknown",
            "address": fault_data.get("crash_location", {}).get("address", "Unknown address"),
            "file": "N/A",
            "line": "N/A"
        }
        
        # ELF Info takes priority if available
        if elf_info:
            loc["function"] = elf_info.get("name", loc["function"])
            loc["file"] = elf_info.get("file", loc["file"])
            loc["line"] = elf_info.get("line", loc["line"])
            
        return loc
