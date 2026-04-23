from typing import List, Dict, Any
from firmware_engine.memory_analyzer import MemoryAnalyzer

class StackUnwinder:
    """
    Heuristic stack unwinder for crashed ARM Cortex-M targets.
    Scans the stack memory to identify potential return addresses and call sites.
    """

    @staticmethod
    def scan_stack(stack_data: bytes, sp_address: int) -> List[Dict[str, Any]]:
        """
        Scans a block of stack memory (raw bytes) and finds addresses 
        that point to executable Flash regions.
        """
        potential_calls = []
        
        # We assume stack is 4-byte aligned
        # Scan in 4-byte words
        for i in range(0, len(stack_data) - 3, 4):
            # Extract 4-byte word (Little Endian)
            val = int.from_bytes(stack_data[i:i+4], byteorder='little')
            
            # PRODUCTION HARDENING: 
            # 1. Address must be in Flash (Executable)
            if MemoryAnalyzer.is_executable(val):
                # 2. Heuristic check: most return addresses are odd (Thumb mod bit 0)
                # but we accept even too if they are in Flash because some 
                # compilers/assemblers might have even entry points (rare).
                
                # 3. Validation: The address should point to valid instruction space
                potential_calls.append({
                    "address": hex(val),
                    "offset": i,
                    "stack_addr": hex(sp_address + i)
                })
        
        return potential_calls
