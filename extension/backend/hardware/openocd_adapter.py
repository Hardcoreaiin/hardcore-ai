import subprocess
import time
import os
import shutil
import glob
from typing import Optional
from hardware.base_adapter import HardwareAdapter

class OpenOCDAdapter(HardwareAdapter):
    """
    Manages the OpenOCD process and provides basic target control.
    """

    def __init__(self, binary_path: str = "openocd"):
        self.binary_path = self._find_binary(binary_path)
        self.process: Optional[subprocess.Popen] = None
        self._connected = False

    def _find_binary(self, name: str) -> str:
        # 1. Check if in PATH
        path = shutil.which(name)
        if path: return path

        if os.name == 'nt': # Windows specific discovery
            name_ext = f"{name}.exe"
            user_profile = os.environ.get('USERPROFILE', '')
            program_files = os.environ.get('ProgramFiles', 'C:\\Program Files')
            
            # Common search patterns
            search_patterns = [
                os.path.join(user_profile, "Downloads", "**", name_ext),
                "C:\\ST\\**\\" + name_ext,
                os.path.join(program_files, "**", name_ext),
                os.path.join(user_profile, "AppData", "Local", "**", name_ext)
            ]
            
            print(f"[Discovery] Searching for {name_ext} in common folders...")
            for pattern in search_patterns:
                matches = glob.glob(pattern, recursive=True)
                if matches:
                    found = matches[0]
                    print(f"[Discovery] Found {name} at: {found}")
                    return found

        return name # Return original and hope for the best

    def connect(self, interface: str, target: str) -> bool:
        """
        Launch OpenOCD with the specified interface and target configs.
        """
        # Auto-resolve script path relative to bin/openocd.exe
        scripts_path = None
        bin_dir = os.path.dirname(self.binary_path)
        
        # Check if scripts folder is in common locations relative to binary
        potential_scripts = [
            os.path.abspath(os.path.join(bin_dir, "..", "scripts")),
            os.path.abspath(os.path.join(bin_dir, "..", "share", "openocd", "scripts")),
            os.path.abspath(os.path.join(bin_dir, "scripts"))
        ]
        
        # Broad search in parent of the plugin folder (for split distributions like STM32CubeIDE)
        plugin_root = os.path.abspath(os.path.join(bin_dir, "..", "..", ".."))
        if os.path.isdir(plugin_root):
             # Search for st_scripts or scripts folder in adjacent plugins
             for pattern in ["**/st_scripts", "**/openocd/scripts"]:
                 matches = glob.glob(os.path.join(plugin_root, pattern), recursive=True)
                 if matches:
                     potential_scripts.append(matches[0])

        for p in potential_scripts:
            if os.path.isdir(p) and (os.path.isdir(os.path.join(p, "interface")) or os.path.isdir(os.path.join(p, "target"))):
                scripts_path = p
                print(f"[Discovery] Using OpenOCD script library: {p}")
                break

        cmd = [self.binary_path]
        if scripts_path:
            cmd.extend(["-s", scripts_path])
        
        cmd.extend(["-f", interface, "-f", target])
        
        try:
            # Start OpenOCD in the background
            self.process = subprocess.Popen(
                cmd,
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
                text=True
            )
            
            # Wait a few seconds for it to initialize
            time.sleep(2)
            if self.process.poll() is None:
                self._connected = True
                return True
            else:
                _, stderr = self.process.communicate()
                print(f"[OpenOCD] CRITICAL ERROR: {stderr.strip()}")
                return False
        except Exception as e:
            print(f"[OpenOCD] Startup exception: {e}")
            return False

    def disconnect(self) -> None:
        if self.process:
            self.process.terminate()
            self.process.wait(timeout=2)
            self.process = None
        self._connected = False

    def is_connected(self) -> bool:
        return self._connected and self.process is not None and self.process.poll() is None

    def get_state(self) -> str:
        # OpenOCD state is usually queried via GDB or Telnet.
        # This adapter primarily manages the process lifecycle.
        return "running" if self.is_connected() else "unknown"

    def read_registers(self, regs: list) -> dict:
        # registers are typically read via GDBAdapter
        return {}

    def read_memory(self, address: int, size: int) -> bytes:
        return b""
