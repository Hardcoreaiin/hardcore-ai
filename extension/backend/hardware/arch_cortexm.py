import os
from typing import Dict, Any
from hardware.arch_adapter import ArchitectureAdapter
from firmware_engine.fault_decoder.decoder import HardFaultDecoder

class CortexMAdapter(ArchitectureAdapter):
    """
    Adapter for ARM Cortex-M architecture.
    """

    def __init__(self, rules_path: str = None):
        if rules_path is None:
            rules_path = os.path.join(os.path.dirname(__file__), "..", "rule_engine", "fault_rules.json")
        self.decoder = HardFaultDecoder(rules_path)

    def detect(self, gdb) -> bool:
        arch_output = gdb.execute("show architecture").lower()
        # Broad detection for any ARM-related strings that GDB might return
        keywords = ["arm", "thumb", "cortex", "m3", "m4", "m7", "v6-m", "v7-m", "v8-m"]
        return any(kw in arch_output for kw in keywords)

    def capture_registers(self, gdb) -> Dict[str, Any]:
        """
        Capture SCB and core registers using high-precision methods.
        """
        # Capture the hardware fault registers (CFSR, HFSR, etc)
        results = gdb.read_scb_fault_registers()
        
        # Add core registers
        results["pc"] = gdb.get_reg("pc")
        results["lr"] = gdb.get_reg("lr")
        results["sp"] = gdb.get_reg("sp")
        
        return results

    def analyze(self, data: Dict[str, Any]) -> Dict[str, Any]:
        """
        Decode the fault registers using the HardFaultDecoder.
        """
        cfsr = data.get("cfsr", 0)
        hfsr = data.get("hfsr", 0)
        mmfar = data.get("mmfar", 0)
        bfar = data.get("bfar", 0)
        
        decoded = self.decoder.decode(cfsr, hfsr, mmfar, bfar)
        
        # Merge in core registers
        decoded["pc"] = hex(data.get("pc", 0)) if data.get("pc") is not None else "Unknown"
        decoded["lr"] = hex(data.get("lr", 0)) if data.get("lr") is not None else "Unknown"
        decoded["sp"] = hex(data.get("sp", 0)) if data.get("sp") is not None else "Unknown"
        
        # Add crash location for DiagnosticEngine
        decoded["crash_location"] = {"address": decoded["pc"]}
        
        return decoded
