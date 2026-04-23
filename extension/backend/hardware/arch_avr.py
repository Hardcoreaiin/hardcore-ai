from typing import Dict, Any
from hardware.arch_adapter import ArchitectureAdapter

class AVRAdapter(ArchitectureAdapter):
    """
    Adapter for AVR architecture (ATmega, etc.).
    Note: AVR typically doesn't have fault registers. 
    Analysis is based on PC and stack state.
    """

    def detect(self, gdb) -> bool:
        arch = gdb.execute("show architecture")
        return "avr" in arch.lower()

    def capture_registers(self, gdb) -> Dict[str, Any]:
        """
        Capture basic registers for AVR.
        """
        return {
            "pc": gdb.get_reg("pc"),
            "sp": gdb.get_reg("sp"),
            "sreg": gdb.get_reg("sreg"),
            "r0": gdb.get_reg("r0")
        }

    def analyze(self, data: Dict[str, Any]) -> Dict[str, Any]:
        """
        Heuristic analysis for AVR 'crashes'.
        """
        pc = data.get("pc", 0)
        sp = data.get("sp", 0)
        
        fault_type = "Halted"
        details = []
        likely_scenario = "Target halted at a specific location."
        
        if pc == 0:
            fault_type = "Reset / Jump to zero"
            details.append({
                "name": "RESET_VECTOR", 
                "description": "PC is at address 0. Likely a software reset or jump to NULL.",
                "register": "pc"
            })
            likely_scenario = "The processor reset or jumped to the NULL address (0x0000)."
        elif sp == 0:
            fault_type = "Stack Error"
            details.append({
                "name": "INVALID_SP",
                "description": "Stack pointer is at 0. Likely stack overflow or uninitialized SP.",
                "register": "sp"
            })
            likely_scenario = "The stack pointer reached 0, indicating a potential stack collision with the beginning of SRAM."

        results = {
            "fault_type": fault_type,
            "details": details,
            "address_info": {},
            "sp": hex(sp) if sp is not None else "Unknown",
            "pc": hex(pc) if pc is not None else "Unknown",
            "crash_location": {"address": hex(pc) if pc is not None else "Unknown"},
            "likely_scenario": likely_scenario
        }
        
        return results
