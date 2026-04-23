import sys
import os
import json

# Add parent to path
sys.path.append(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from cli.diagnostic_engine import DiagnosticEngine

def test_stack_smash_scenario():
    print("Testing 'Stack Smash' Scenario Logic...")
    engine = DiagnosticEngine()
    
    # Mock fault data (UNDEFINSTR + PC in RAM)
    fault_data = {
        "fault_type": "UsageFault",
        "details": [{"name": "UNDEFINSTR", "register": "UFSR", "description": "Undefined Instruction"}],
        "address_info": {},
        "crash_location": {"address": "0x20001234"} # RAM region
    }
    
    report = engine.analyze(fault_data)
    
    # Verify expectations
    assert "RAM" in report["likely_scenario"]
    assert "Undefined Instruction" in report["fault_type"]
    assert report["memory_analysis"]["region"] == "SRAM"
    
    print("\n[VERIFIED] Likely Scenario: " + report["likely_scenario"])
    print("[VERIFIED] Fix Checklist: " + str(report["fix"]))
    print("\n--- TEST PASSED ---")

def test_null_pointer_scenario():
    print("\nTesting 'Null Pointer' Scenario Logic...")
    engine = DiagnosticEngine()
    
    # Mock fault data (PRECISERR + Address 0x0)
    fault_data = {
        "fault_type": "BusFault",
        "details": [{"name": "PRECISERR", "register": "BFSR", "description": "Precise Data Access Error"}],
        "address_info": {"BusFault Address": "0x00000000"},
        "crash_location": {"address": "0x08001000"} # Valid flash
    }
    
    report = engine.analyze(fault_data)
    
    # Verify expectations
    assert "Null Pointer" in report["likely_scenario"]
    assert report["memory_analysis"]["region"] == "FLASH"
    
    print("\n[VERIFIED] Likely Scenario: " + report["likely_scenario"])
    print("[VERIFIED] Fix Checklist: " + str(report["fix"]))
    print("\n--- TEST PASSED ---")

if __name__ == "__main__":
    test_stack_smash_scenario()
    test_null_pointer_scenario()
