from typing import Dict, Any
from hardware.arch_adapter import ArchitectureAdapter

class RISCVAdapter(ArchitectureAdapter):
    """
    Adapter for RISC-V architecture.
    """

    def detect(self, gdb) -> bool:
        arch = gdb.execute("show architecture")
        return "riscv" in arch.lower()

    def capture_registers(self, gdb) -> Dict[str, Any]:
        """
        Capture CSRs and core registers.
        """
        return {
            "mcause": gdb.get_reg("mcause"),
            "mepc": gdb.get_reg("mepc"),
            "mtval": gdb.get_reg("mtval"),
            "sp": gdb.get_reg("sp"),
            "pc": gdb.get_reg("pc") # Usually same as mepc if halted at fault
        }

    def analyze(self, data: Dict[str, Any]) -> Dict[str, Any]:
        """
        Decode RISC-V mcause into human-readable format.
        """
        mcause = data.get("mcause", 0)
        mtval = data.get("mtval", 0)
        mepc = data.get("mepc", 0)
        
        # Mapping mcause to common names for DiagnosticEngine compatibility
        causes = {
            0: ("Instruction address misaligned", "MISALIGNED_INSTR"),
            1: ("Instruction access fault", "INSTR_ACCESS_FAULT"),
            2: ("Illegal instruction", "UNDEFINSTR"), # Map to UNDEFINSTR for compat
            3: ("Breakpoint", "BREAKPOINT"),
            4: ("Load address misaligned", "MISALIGNED_LOAD"),
            5: ("Load access fault", "PRECISERR"), # Map to PRECISERR for compat
            6: ("Store/AMO address misaligned", "MISALIGNED_STORE"),
            7: ("Store/AMO access fault", "PRECISERR"), # Map to PRECISERR for compat
        }
        
        cause_info = causes.get(mcause & 0x7FFFFFFF, ("Unknown Exception", "UNKNOWN"))
        is_interrupt = bool(mcause & 0x80000000)
        
        results = {
            "fault_type": f"RISC-V {'Interrupt' if is_interrupt else 'Exception'} ({cause_info[0]})",
            "details": [
                {
                    "name": cause_info[1],
                    "description": f"mcause: {hex(mcause)} ({cause_info[0]})",
                    "register": "mcause"
                }
            ],
            "address_info": {}
        }
        
        if mtval:
            results["address_info"]["Fault Address (mtval)"] = hex(mtval)
            # For DiagnosticEngine compatibility (it looks for "BusFault Address")
            results["address_info"]["BusFault Address"] = hex(mtval)
            
        results["sp"] = hex(data.get("sp", 0)) if data.get("sp") is not None else "Unknown"
        results["pc"] = hex(mepc) if mepc is not None else "Unknown"
        results["crash_location"] = {"address": results["pc"]}
        
        return results
