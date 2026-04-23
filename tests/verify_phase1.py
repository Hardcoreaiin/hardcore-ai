import subprocess
import json
import sys
import os

def run_cli(args):
    cli_path = os.path.join(os.path.dirname(__file__), "..", "cli", "hc_fault.py")
    cmd = [sys.executable, cli_path] + args
    result = subprocess.run(cmd, capture_output=True, text=True)
    return result

def test_usage_fault():
    print("Testing UsageFault: UNDEFINSTR...")
    # CFSR = 0x00010000 (UNDEFINSTR bit in UFSR)
    # HFSR = 0x40000000 (FORCED bit)
    args = ["--cfsr", "0x00010000", "--hfsr", "0x40000000", "--json"]
    res = run_cli(args)
    if res.returncode != 0:
        print(f"CLI Error: {res.stderr}")
        return False
    
    data = json.loads(res.stdout)
    assert data["fault_type"] == "Escalated to HardFault"
    assert any(d["name"] == "UNDEFINSTR" for d in data["details"])
    print("UsageFault test PASSED.")
    return True

def test_bus_fault_with_address():
    print("Testing BusFault with address...")
    # CFSR = 0x00008200 (PRECISERR + BFARVALID in BFSR)
    # HFSR = 0x40000000 (FORCED bit)
    # BFAR = 0xDEADBEEF
    args = ["--cfsr", "0x00008200", "--hfsr", "0x40000000", "--bfar", "0xDEADBEEF", "--json"]
    res = run_cli(args)
    if res.returncode != 0:
        print(f"CLI Error: {res.stderr}")
        return False
    
    data = json.loads(res.stdout)
    assert any(d["name"] == "PRECISERR" for d in data["details"])
    assert data["address_info"]["BusFault Address"] == "0xdeadbeef"
    print("BusFault test PASSED.")
    return True

if __name__ == "__main__":
    success = True
    success &= test_usage_fault()
    success &= test_bus_fault_with_address()
    
    if success:
        print("\nAll Phase 1 CLI tests PASSED.")
        sys.exit(0)
    else:
        print("\nSome tests FAILED.")
        sys.exit(1)
