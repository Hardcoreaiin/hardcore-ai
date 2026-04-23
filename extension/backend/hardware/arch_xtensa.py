from typing import Dict, Any
from hardware.arch_adapter import ArchitectureAdapter

class XtensaAdapter(ArchitectureAdapter):
    """
    Adapter for Xtensa architecture (ESP32).
    """

    def detect(self, gdb) -> bool:
        arch = gdb.execute("show architecture")
        return "xtensa" in arch.lower()

    def capture_registers(self, gdb) -> Dict[str, Any]:
        """
        Capture exception registers for Xtensa.
        """
        return {
            "exccause": gdb.get_reg("exccause"),
            "excvaddr": gdb.get_reg("excvaddr"),
            "epc1": gdb.get_reg("epc1"),
            "sp": gdb.get_reg("a1"), # A1 is the stack pointer for Xtensa
            "pc": gdb.get_reg("pc")
        }

    def analyze(self, data: Dict[str, Any]) -> Dict[str, Any]:
        """
        Decode Xtensa exccause.
        """
        exccause = data.get("exccause", 0)
        excvaddr = data.get("excvaddr", 0)
        epc1 = data.get("epc1", 0)
        
        # Xtensa Exception Cause Mapping (Simplified)
        causes = {
            0: ("IllegalInstruction", "UNDEFINSTR"),
            1: ("Syscall", "SYSCALL"),
            2: ("InstructionFetchError", "PRECISERR"),
            3: ("LoadStoreError", "PRECISERR"),
            9: ("PrivilegedInstruction", "PRIVILEGED"),
            12: ("InstrFetchProhibited", "ACCESSRIGHTS"),
            13: ("LoadProhibited", "PRECISERR"),
            14: ("StoreProhibited", "PRECISERR"),
            28: ("LoadStoreAlignment", "INVSTATE"), # Closest match for unaligned
            29: ("LoadStoreAlignment", "INVSTATE")
        }
        
        cause_info = causes.get(exccause, ("Unknown Exception", "UNKNOWN"))
        
        results = {
            "fault_type": f"Xtensa Exception ({cause_info[0]})",
            "details": [
                {
                    "name": cause_info[1],
                    "description": f"exccause: {exccause} ({cause_info[0]})",
                    "register": "exccause"
                }
            ],
            "address_info": {}
        }
        
        if excvaddr is not None:
            results["address_info"]["Exception Address (excvaddr)"] = hex(excvaddr)
            results["address_info"]["BusFault Address"] = hex(excvaddr) # Compat
            
        results["sp"] = hex(data.get("sp", 0)) if data.get("sp") is not None else "Unknown"
        results["pc"] = hex(epc1) if epc1 is not None else "Unknown"
        results["crash_location"] = {"address": results["pc"]}
        
        return results
