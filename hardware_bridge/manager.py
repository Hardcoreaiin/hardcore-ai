"""
Hardware Device Manager
Central coordinator for hardware interactions.
"""

from typing import Dict, Any, List, Optional
from .scanner import HardwareScanner
from .flasher import HardwareFlasher
import os

class DeviceManager:
    """
    Manages connected hardware devices.
    """
    
    def __init__(self):
        self.scanner = HardwareScanner()
        self.flasher = HardwareFlasher()
        self.active_port: Optional[str] = None
        self.last_scan_result: List[Dict[str, Any]] = []
        
    def scan_devices(self) -> List[Dict[str, Any]]:
        """Scan and return list of devices."""
        self.last_scan_result = self.scanner.scan_ports()
        
        # Auto-select if we don't have an active one, or if active one is gone
        self._validate_active_port()
        
        return self.last_scan_result
        
    def select_port(self, port_name: str) -> bool:
        """Manually select a port."""
        # Validate port exists in last scan
        valid = any(d["port"] == port_name for d in self.last_scan_result)
        # Or scan again to be sure
        if not valid:
            self.scan_devices()
            valid = any(d["port"] == port_name for d in self.last_scan_result)
            
        if valid:
            self.active_port = port_name
            return True
        return False
        
    def get_status(self) -> Dict[str, Any]:
        """Get current connection status."""
        return {
            "connected": self.active_port is not None,
            "port": self.active_port,
            "device_count": len(self.last_scan_result)
        }

    def flash_firmware(self, firmware_path: str) -> Dict[str, Any]:
        """
        Trigger flash on active port.
        """
        if not self.active_port:
            return {"success": False, "error": "No device connected"}
            
        if not os.path.exists(firmware_path):
            return {"success": False, "error": f"Firmware path not found: {firmware_path}"}
            
        # Call flasher
        result = self.flasher.flash_firmware(self.active_port, firmware_path)
        return result

    def _validate_active_port(self):
        """Check if active port is still in the list."""
        if self.active_port:
            still_there = any(d["port"] == self.active_port for d in self.last_scan_result)
            if not still_there:
                self.active_port = None
