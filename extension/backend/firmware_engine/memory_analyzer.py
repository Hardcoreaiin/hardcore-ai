from typing import Dict, Optional

class MemoryAnalyzer:
    """
    Expert memory region analyzer for ARM Cortex-M architecture.
    Determines if memory access or execution is within valid boundaries.
    """
    
    # Standard Cortex-M Memory Map
    REGIONS = {
        "FLASH":    (0x08000000, 0x08FFFFFF, "Main Flash Memory (Code)"),
        "SRAM":     (0x20000000, 0x2007FFFF, "Static RAM (Data/Stack)"),
        "PERIPH":   (0x40000000, 0x5FFFFFFF, "Peripheral Registers"),
        "PPB":      (0xE0000000, 0xE00FFFFF, "Private Peripheral Bus (SCB/NVIC/MPU)"),
        "SYSTEM":   (0xF0000000, 0xFFFFFFFF, "System / Vendor Space")
    }

    @staticmethod
    def analyze_address(address: int) -> Dict[str, str]:
        """
        Determines the region and validity of a memory address.
        """
        for region, (start, end, desc) in MemoryAnalyzer.REGIONS.items():
            if start <= address <= end:
                return {
                    "region": region,
                    "description": desc,
                    "valid": "PASS" if region in ["FLASH", "SRAM"] else "WARN",
                    "executable": "YES" if region == "FLASH" else "DATA-ONLY"
                }
        
        return {
            "region": "INVALID",
            "description": "Outside mapped memory space",
            "valid": "FAIL",
            "executable": "NO"
        }

    @staticmethod
    def is_executable(address: int) -> bool:
        """ Returns True if the address is in a known executable region (Flash). """
        info = MemoryAnalyzer.analyze_address(address)
        return info["executable"] == "YES"
