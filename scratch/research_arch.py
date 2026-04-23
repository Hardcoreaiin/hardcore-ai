import sys
import os
import time
from hardware.gdb_adapter import GDBAdapter

def check_arch():
    gdb = GDBAdapter()
    print("Connecting to OpenOCD...")
    # Using the same target as the user
    if gdb.connect(target_remote="localhost:3333"):
        print("Connected.")
        
        # Test 1: Original detection command
        res = gdb.execute("show architecture")
        print(f"DEBUG: 'show architecture' raw response: '{res}'")
        
        # Test 2: Checking multiple commands
        res_info = gdb.execute("info target")
        print(f"DEBUG: 'info target' raw response: '{res_info}'")
        
        gdb.disconnect()
    else:
        print("Failed to connect.")

if __name__ == "__main__":
    check_arch()
