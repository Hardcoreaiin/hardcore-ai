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
            
            # Optimized search patterns (Limited depth for speed)
            search_patterns = [
                os.path.join(user_profile, "Downloads", "*", name_ext),
                os.path.join(user_profile, "Downloads", "*", "*", name_ext),
                "C:\\ST\\" + name_ext,
                "C:\\ST\\**\\" + name_ext, # ST folder is usually safe for recursive
                os.path.join(program_files, "OpenOCD*", "bin", name_ext),
                os.path.join(user_profile, "AppData", "Local", "Programs", "**", name_ext)
            ]
            
            print(f"[Discovery] Searching for {name_ext} (Fast scan)...")
            for pattern in search_patterns:
                recursive = "**" in pattern
                matches = glob.glob(pattern, recursive=recursive)
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
        
        # Broad search in parent of the plugin folder (Optimized for STM32CubeIDE)
        plugin_root = os.path.abspath(os.path.join(bin_dir, "..", "..", ".."))
        if os.path.isdir(plugin_root):
             # Fast Scan: Only look in relevant ST plugin folders
             for folder in os.listdir(plugin_root):
                 if "mcu.debug.openocd" in folder:
                     full_path = os.path.join(plugin_root, folder)
                     # Check common locations within the specific plugin
                     for sub in ["resources/openocd/st_scripts", "resources/openocd/scripts"]:
                         p = os.path.join(full_path, sub)
                         if os.path.isdir(p):
                             potential_scripts.append(p)

        for p in potential_scripts:
            if os.path.isdir(p) and (os.path.isdir(os.path.join(p, "interface")) or os.path.isdir(os.path.join(p, "target"))):
                scripts_path = p
                print(f"[Discovery] Using OpenOCD script library: {p}")
                break

        # Normalize and Quote all paths for Windows Shell
        binary = f'"{os.path.normpath(self.binary_path)}"'
        scripts_arg = f'-s "{os.path.normpath(scripts_path)}"' if scripts_path else ""
        interface_arg = f'-f "{os.path.normpath(interface)}"'
        target_arg = f'-f "{os.path.normpath(target)}"'
        
        # Build the command string for shell execution
        cmd_str = f'{binary} {scripts_arg} {interface_arg} {target_arg}'

        print(f"[OpenOCD] Launching (Shell Mode): {cmd_str}")
        try:
            self.process = subprocess.Popen(
                cmd_str,
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
                text=True,
                shell=True,
                bufsize=0,
                creationflags=subprocess.CREATE_NO_WINDOW if os.name == 'nt' else 0
            )
            
            # Brief wait to see if it starts
            time.sleep(1.0)
            if self.process.poll() is not None:
                _, stderr = self.process.communicate()
                print(f"[OpenOCD] CRITICAL ERROR: {stderr}")
                return False
                
            return True
        except Exception as e:
            print(f"Failed to start OpenOCD: {e}")
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
