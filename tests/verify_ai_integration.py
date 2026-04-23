import sys
import os
import asyncio
import json
import threading
import time
from unittest.mock import MagicMock, AsyncMock

# Add project root to path
sys.path.append(os.path.abspath(os.path.join(os.path.dirname(__file__), '..')))

from hardware.orchestrator import HardwareOrchestrator
from learning_engine.ai_crash_analyzer import AICrashAnalyzer

async def verify_ai_flow():
    print("Testing AI Intelligence Layer Integration...")
    
    # 1. Mock the AI Analyzer to avoid real API calls
    mock_ai_result = {
        "root_cause_explanation": "Test explanation: The peripheral clock is disabled.",
        "confidence": "high",
        "fix_steps": ["Enable RCC clock", "Retry access"],
        "bug_pattern": "Clock Gating Issue",
        "risk_level": "high"
    }
    
    # 2. Setup Orchestrator with mock config
    config = {
        "interface": "mock_interface",
        "target": "mock_target",
        "elf_path": "mock.elf"
    }
    
    orchestrator = HardwareOrchestrator(config)
    orchestrator.ai_analyzer.analyze = AsyncMock(return_value=mock_ai_result)
    
    # 3. Simulate a fault discovery
    # We mock the arch and GDB parts
    orchestrator.arch = MagicMock()
    orchestrator.arch.capture_registers.return_value = {"pc": 0x40021000, "lr": 0x08001234}
    orchestrator.arch.analyze.return_value = {
        "fault_type": "Bus Access Error",
        "details": [{"name": "PRECISERR", "register": "CFSR"}],
        "where": {"address": "0x40021000", "function": "i2c_write", "file": "i2c.c", "line": 45}
    }
    
    # 4. Trigger analysis
    final_report = None
    def on_fault(report):
        nonlocal final_report
        # This will be called twice: once for deterministic, once after AI
        if "ai_analysis" in report:
            final_report = report
            print("Received report with AI analysis!")

    orchestrator.on_fault_detected = on_fault
    orchestrator._perform_auto_analysis()
    
    # Wait for the AI thread to finish
    timeout = 5
    start_time = time.time()
    while final_report is None and time.time() - start_time < timeout:
        time.sleep(0.1)
    
    if final_report and "ai_analysis" in final_report:
        print("\nSUCCESS: AI result merged into report.")
        print(json.dumps(final_report["ai_analysis"], indent=2))
        return True
    else:
        print("\nFAILURE: AI result not found in report.")
        return False

if __name__ == "__main__":
    if asyncio.run(verify_ai_flow()):
        print("\nAI INTEGRATION VERIFIED!")
    else:
        sys.exit(1)
