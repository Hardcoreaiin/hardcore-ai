import sys
import os
from unittest.mock import MagicMock

# Add project root to path
sys.path.append(os.path.abspath(os.path.join(os.path.dirname(__file__), '..')))

from hardware.arch_cortexm import CortexMAdapter
from hardware.arch_riscv import RISCVAdapter
from hardware.arch_xtensa import XtensaAdapter
from hardware.arch_avr import AVRAdapter
from cli.diagnostic_engine import DiagnosticEngine

def test_cortex_m():
    print("Testing Cortex-M Adapter...")
    mock_gdb = MagicMock()
    # Mock show architecture
    mock_gdb.execute.return_value = "The target architecture is set to \"auto\" (currently \"arm\")."
    # Mock registers/memory
    mock_gdb.read_mem.side_effect = lambda addr, size: {
        "0xE000ED28": 0x010000, # CFSR: UNDEFINSTR
        "0xE000ED2C": 0x40000000, # HFSR: FORCED
        "0xE000ED34": 0x0,
        "0xE000ED38": 0x0
    }.get(addr, 0)
    mock_gdb.get_reg.side_effect = lambda name: {
        "pc": 0x08001234,
        "lr": 0x08005678,
        "sp": 0x20001000
    }.get(name, 0)

    adapter = CortexMAdapter()
    assert adapter.detect(mock_gdb) is True
    
    raw = adapter.capture_registers(mock_gdb)
    print(f"Captured: {raw}")
    
    decoded = adapter.analyze(raw)
    print(f"Decoded: {decoded}")
    
    engine = DiagnosticEngine()
    report = engine.analyze(decoded)
    print(f"Report Fault Type: {report['fault_type']}")
    assert "Undefined Instruction" in report['fault_type']
    print("Cortex-M Test Passed!")

def test_riscv():
    print("\nTesting RISC-V Adapter...")
    mock_gdb = MagicMock()
    # Mock show architecture
    mock_gdb.execute.return_value = "The target architecture is set to \"auto\" (currently \"riscv:rv32\")."
    # Mock registers
    mock_gdb.get_reg.side_effect = lambda name: {
        "mcause": 2, # Illegal Instruction
        "mepc": 0x80001234,
        "mtval": 0x0,
        "sp": 0x80010000,
        "pc": 0x80001234
    }.get(name, 0)

    adapter = RISCVAdapter()
    assert adapter.detect(mock_gdb) is True
    
    raw = adapter.capture_registers(mock_gdb)
    print(f"Captured: {raw}")
    
    decoded = adapter.analyze(raw)
    print(f"Decoded: {decoded}")
    
    engine = DiagnosticEngine()
    report = engine.analyze(decoded)
    print(f"Report Fault Type: {report['fault_type']}")
    # Now respects the RISC-V specific fault type
    assert "RISC-V Exception" in report['fault_type'] and "Illegal instruction" in report['fault_type']
    print("RISC-V Test Passed!")

def test_xtensa():
    print("\nTesting Xtensa Adapter...")
    mock_gdb = MagicMock()
    # Mock show architecture
    mock_gdb.execute.return_value = "The target architecture is set to \"auto\" (currently \"xtensa\")."
    # Mock registers
    mock_gdb.get_reg.side_effect = lambda name: {
        "exccause": 0, # Illegal Instruction
        "excvaddr": 0,
        "epc1": 0x40001234,
        "a1": 0x3fffe000,
        "pc": 0x40001234
    }.get(name, 0)

    adapter = XtensaAdapter()
    assert adapter.detect(mock_gdb) is True
    
    raw = adapter.capture_registers(mock_gdb)
    print(f"Captured: {raw}")
    
    decoded = adapter.analyze(raw)
    print(f"Decoded: {decoded}")
    
    engine = DiagnosticEngine()
    report = engine.analyze(decoded)
    print(f"Report Fault Type: {report['fault_type']}")
    # Now respects the Xtensa specific fault type
    assert "Xtensa Exception" in report['fault_type'] and "IllegalInstruction" in report['fault_type']
    print("Xtensa Test Passed!")

def test_avr():
    print("\nTesting AVR Adapter...")
    mock_gdb = MagicMock()
    # Mock show architecture
    mock_gdb.execute.return_value = "The target architecture is set to \"auto\" (currently \"avr\")."
    # Mock registers
    mock_gdb.get_reg.side_effect = lambda name: {
        "pc": 0, # Reset Vector
        "sp": 0x0800,
        "sreg": 0,
        "r0": 0
    }.get(name, 0)

    adapter = AVRAdapter()
    assert adapter.detect(mock_gdb) is True
    
    raw = adapter.capture_registers(mock_gdb)
    print(f"Captured: {raw}")
    
    decoded = adapter.analyze(raw)
    print(f"Decoded: {decoded}")
    
    engine = DiagnosticEngine()
    report = engine.analyze(decoded)
    print(f"Report Fault Type: {report['fault_type']}")
    assert "Reset / Jump to zero" in report['fault_type']
    print("AVR Test Passed!")

if __name__ == "__main__":
    try:
        test_cortex_m()
        test_riscv()
        test_xtensa()
        test_avr()
        print("\nALL ARCHITECTURE ADAPTER TESTS PASSED!")
    except Exception as e:
        print(f"\nTEST FAILED: {e}")
        import traceback
        traceback.print_exc()
        sys.exit(1)
