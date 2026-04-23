import subprocess
import json
import time
import os
import re

GDB_PATH = "arm-none-eabi-gdb"
ELF_PATH = r"C:\Users\shriv\STM32CubeIDE\workspace_1.19.0\hctest\Debug\hctest.elf"

# ✅ Always write to HCv2 root
BASE_DIR = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
AUTO_FAULT_FILE = os.path.join(BASE_DIR, "auto_fault.json")


def run_gdb_command(cmds):
    process = subprocess.Popen(
        [GDB_PATH, ELF_PATH],
        stdin=subprocess.PIPE,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True
    )

    output = ""

    for cmd in cmds:
        process.stdin.write(cmd + "\n")
        process.stdin.flush()
        time.sleep(0.4)   # 🔥 CRITICAL: allow MCU + GDB sync

    process.stdin.write("quit\n")
    process.stdin.flush()

    output, _ = process.communicate()
    return output


# ✅ Correct parsing (RIGHT SIDE ONLY)
def extract_register(output, address):
    address = address.lower()

    for line in output.splitlines():
        line_clean = line.strip().lower()

        if line_clean.startswith(address + ":"):
            parts = line_clean.split(":")
            if len(parts) >= 2:
                value = parts[1].strip()

                match = re.search(r'0x[0-9a-fA-F]+', value)
                if match:
                    return match.group(0)

    return "0x0"


# ✅ Robust PC extraction
def extract_pc(output):
    for line in output.splitlines():
        line_clean = line.strip()

        if line_clean.startswith("pc"):
            parts = line_clean.split()
            for part in parts:
                if part.startswith("0x"):
                    return part

    return "0x0"


def capture_fault():
    cmds = [
        "target remote localhost:3333",

        # 🔥 Reset cleanly
        "monitor reset halt",

        # 🔥 Run firmware (will crash)
        "continue",

        # 🔥 Give time for crash to happen
        "monitor sleep 500",

        # 🔥 Halt AFTER crash
        "monitor halt",

        # 🔥 Read fault registers
        "x/1wx 0xE000ED28",
        "x/1wx 0xE000ED2C",
        "x/1wx 0xE000ED34",
        "x/1wx 0xE000ED38",

        # 🔥 Read PC
        "info registers"
    ]

    output = run_gdb_command(cmds)

    print("\n===== RAW GDB OUTPUT =====")
    print(output)
    print("=========================\n")

    cfsr = extract_register(output, "0xe000ed28")
    hfsr = extract_register(output, "0xe000ed2c")
    mmfar = extract_register(output, "0xe000ed34")
    bfar = extract_register(output, "0xe000ed38")
    pc = extract_pc(output)

    return {
        "cfsr": cfsr,
        "hfsr": hfsr,
        "mmfar": mmfar,
        "bfar": bfar,
        "pc": pc
    }


def main():
    print("🔍 HARDCOREAI Live Monitor Started...")

    while True:
        try:
            fault = capture_fault()

            if fault["cfsr"] not in ["0x0", "0x00000000"]:
                print("💥 Fault detected!")
                print("Captured:", fault)

                with open(AUTO_FAULT_FILE, "w") as f:
                    json.dump(fault, f, indent=2)

                print(f"📄 auto_fault.json written to: {AUTO_FAULT_FILE}")
                break

        except Exception as e:
            print("Error:", e)

        time.sleep(2)


if __name__ == "__main__":
    main()