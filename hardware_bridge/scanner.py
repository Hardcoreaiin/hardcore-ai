"""
Hardware Scanner - Device Detection
Scans for connected ESP32 devices via Serial/COM ports.
"""

import sys
import serial.tools.list_ports
from typing import List, Dict, Any

class HardwareScanner:
    """Scans for ESP32 and related microcontrollers."""
    
    # Common USB-Serial Drivers for ESP32
    ESP32_DRIVERS = [
        "CP210", "CH340", "FTDI", "USB Serial", "Silicon Labs"
    ]
    
    def scan_ports(self) -> List[Dict[str, Any]]:
        """
        List all available serial ports and potential ESP32s.
        
        Returns:
            List of dicts: [{"port": "COM3", "desc": "CP2102...", "is_esp32": True}]
        """
        devices = []
        try:
            ports = serial.tools.list_ports.comports()
            
            for port in ports:
                is_esp = any(driver in port.description for driver in self.ESP32_DRIVERS)
                
                devices.append({
                    "port": port.device,
                    "description": port.description,
                    "hwid": port.hwid,
                    "is_likely_esp32": is_esp
                })
                
        except Exception as e:
            print(f"[Scanner] Error scanning ports: {e}")
            
        return devices

    def detect_esp32(self) -> str:
        """
        Auto-detect the most likely ESP32 port.
        Returns first likely ESP32 port or None.
        """
        ports = self.scan_ports()
        for p in ports:
            if p["is_likely_esp32"]:
                return p["port"]
        return None
