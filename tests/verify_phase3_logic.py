import sys
import os
import time

# Add paths
sys.path.append(os.path.join(os.path.dirname(__file__), ".."))

from hardware.orchestrator import HardwareOrchestrator

class MockGDB:
    def __init__(self):
        self._halted = False
    def get_state(self):
        return "halted" if self._halted else "running"
    def read_registers(self, regs):
        return {
            "pc": 0x08000100,
            "cfsr": 0x010000,
            "hfsr": 0x40000000
        }
    def connect(self, **kwargs): return True
    def disconnect(self): pass

def test_auto_capture():
    print("Starting Mock Hardware Auto-Capture Test...")
    
    config = {
        "interface": "mock",
        "target": "mock",
        "elf_path": None,
        "uart_port": None
    }
    
    orchestrator = HardwareOrchestrator(config)
    mock_gdb = MockGDB()
    orchestrator.gdb = mock_gdb # Inject mock
    
    reports = []
    def on_fault(report):
        reports.append(report)
        print(f"SUCCESS: Auto-captured fault: {report['fault_type']}")

    orchestrator.on_fault_detected = on_fault
    
    # Simulate monitor loop manually or start it
    orchestrator.running = True
    
    print("Triggering mock halt...")
    mock_gdb._halted = True
    
    # The monitor loop in orchestrator.py runs in a thread
    # Let's see if it picks it up
    orchestrator._monitor_loop() 
    
    if len(reports) > 0:
        print("Test Passed: Orchestrator detected halt and analyzed fault.")
    else:
        print("Test Failed: No report generated.")

if __name__ == "__main__":
    try:
        test_auto_capture()
    except Exception as e:
        print(f"Test Error: {e}")
