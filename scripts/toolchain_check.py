import subprocess
import sys
import os

def check_command(cmd):
    try:
        startupinfo = None
        if os.name == 'nt':
            startupinfo = subprocess.STARTUPINFO()
            startupinfo.dwFlags |= subprocess.STARTF_USESHOWWINDOW
        
        cmd_path = cmd
        if cmd == "openocd":
            cmd_path = r"C:\Users\shriv\Downloads\xpack-openocd-0.12.0-7-win32-x64\xpack-openocd-0.12.0-7\bin\openocd.exe"
        
        result = subprocess.run([cmd_path, "--version"], capture_output=True, text=True, startupinfo=startupinfo)
        ver_info = result.stdout.strip() or result.stderr.strip()
        if ver_info:
            print(f"[OK] {cmd} found: {ver_info.splitlines()[0]}")
        else:
            print(f"[OK] {cmd} found (but version info is empty)")
        return True
    except FileNotFoundError:
        print(f"[ERROR] {cmd} not found in PATH.")
        return False
    except Exception as e:
        print(f"[ERROR] Error checking {cmd}: {e}")
        return False

def main():
    print("--- HARDCOREAI Hardware Toolchain Diagnostic ---")
    
    python_version = sys.version.splitlines()[0]
    print(f"Python Version: {python_version}")
    
    ocd_ok = check_command("openocd")
    gdb_ok = check_command("arm-none-eabi-gdb")
    
    if not ocd_ok:
        print("\nTIP: Install OpenOCD to enable live hardware monitoring.")
    if not gdb_ok:
        print("\nTIP: Install GNU Arm Embedded Toolchain (GDB) for fault analysis.")
        
    if ocd_ok and gdb_ok:
        print("\nSUCCESS: All hardware tools are ready for Phase 3 Live Mode.")
    else:
        print("\nWARNING: Some tools are missing. Live capture may not work.")

if __name__ == "__main__":
    main()
