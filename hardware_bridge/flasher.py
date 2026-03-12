"""
Hardware Flasher - Firmware Upload Logic
Wraps esptool and/or PlatformIO for flashing.
"""

import sys
import os
import subprocess
import time
import threading
from typing import Dict, Any, Optional, Callable

class HardwareFlasher:
    """Handles flashing firmware to ESP32."""
    
    def __init__(self, working_dir: str = "."):
        self.working_dir = working_dir
        
    def flash_firmware(self, port: str, firmware_dir: str, build_cmd: str = "run -t upload", log_callback: Optional[Callable[[str], None]] = None) -> Dict[str, Any]:
        """
        Compile and Flash firmware with optional real-time logging.
        """
        # Ensure upload port is specified
        if "--upload-port" not in build_cmd:
            build_cmd += f" --upload-port {port}"
            
        try:
            # Build execution command with precise path finding
            python_exe = f'"{sys.executable}"'
            
            # PlatformIO global flags must come BEFORE the command (run)
            pio_base = f"{python_exe} -m platformio --no-ansi"
            
            if os.name == 'nt':
                scripts_dir = os.path.join(os.path.dirname(sys.executable), "Scripts")
                direct_pio = os.path.join(scripts_dir, "pio.exe")
                if os.path.exists(direct_pio):
                    pio_base = f'"{direct_pio}" --no-ansi'
            
            # Clean up build_cmd if it mistakenly contains 'pio'
            clean_cmd = build_cmd.strip()
            if clean_cmd.startswith("pio "):
                clean_cmd = clean_cmd[4:].strip()
            elif clean_cmd.startswith("platformio "):
                clean_cmd = clean_cmd[11:].strip()
            
            full_cmd = f"{pio_base} {clean_cmd}"

            if log_callback:
                log_callback(f"[HARDCOREAI] Executing: {full_cmd}")
            
            # Echo to backend console for visibility
            print(f"\n[Flasher] Running Build Command: {full_cmd}")
            print(f"[Flasher] Working Directory: {firmware_dir}\n")

            # Start process with merged streams
            process = subprocess.Popen(
                full_cmd,
                cwd=firmware_dir,
                shell=True,
                stdout=subprocess.PIPE,
                stderr=subprocess.STDOUT,
                text=True,
                bufsize=1, # Line buffered
                env={**os.environ, "PYTHONUNBUFFERED": "1"}
            )
            
            output_accumulated = []
            
            # Capture and stream logs
            while True:
                line = process.stdout.readline()
                if not line and process.poll() is not None:
                    break
                
                if line:
                    output_accumulated.append(line)
                    # Stream to frontend
                    if log_callback:
                        log_callback(line)
                    
                    # ALWAYS stream to backend console for debugging
                    sys.stdout.write(f"PIO: {line}")
                    sys.stdout.flush()
            
            process.wait()
            success = process.returncode == 0
            full_output = "".join(output_accumulated)
            
            if success:
                print(f"\n[Flasher] Build/Flash SUCCESS\n")
            else:
                print(f"\n[Flasher] Build/Flash FAILED (Exit Code: {process.returncode})\n")
            
            return {
                "success": success,
                "output": full_output,
                "error": "" if success else "Build/Flash failed. See logs for details.",
                "command": full_cmd
            }
            
        except Exception as e:
            err_msg = f"Flasher Execution Error: {str(e)}"
            print(f"[Flasher] CRITICAL ERROR: {err_msg}")
            if log_callback: log_callback(f"[ERROR] {err_msg}")
            return {
                "success": False,
                "output": "",
                "error": err_msg,
                "command": full_cmd if 'full_cmd' in locals() else pio_base
            }

    def verify_chip(self, port: str) -> bool:
        """Verify if chip is accessible."""
        try:
            cmd = f"esptool.py --port {port} chip_id"
            result = subprocess.run(cmd, shell=True, capture_output=True, text=True, timeout=5)
            return result.returncode == 0
        except:
            return False
