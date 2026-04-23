from typing import List, Optional
from hardware.arch_adapter import ArchitectureAdapter
from hardware.arch_cortexm import CortexMAdapter
from hardware.arch_riscv import RISCVAdapter
from hardware.arch_xtensa import XtensaAdapter
from hardware.arch_avr import AVRAdapter

adapters: List[ArchitectureAdapter] = [
    CortexMAdapter(),
    RISCVAdapter(),
    XtensaAdapter(),
    AVRAdapter(),
]

def select_adapter(gdb) -> Optional[ArchitectureAdapter]:
    """
    Auto-detect and return the matching architecture adapter.
    """
    for adapter in adapters:
        if adapter.detect(gdb):
            return adapter
    return None
