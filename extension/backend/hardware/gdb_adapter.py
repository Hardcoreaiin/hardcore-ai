import subprocess
import re
import time
import threading
import queue
import os
import shutil
import glob
from typing import Dict, List, Optional, Any, Callable
from hardware.base_adapter import HardwareAdapter

class GDBAdapter(HardwareAdapter):
    """
    Production-grade GDB/MI (Machine Interface) Adapter.
    Uses an asynchronous reader thread to handle persistent sessions and 
    real-time event notifications (e.g., target halts).
    """

    def __init__(self, binary_path: str = "arm-none-eabi-gdb"):
        self.binary_path = self._find_binary(binary_path)
        self.process: Optional[subprocess.Popen] = None
        self._connected = False
        
        # Async handling
        self._reader_thread: Optional[threading.Thread] = None
        self._is_running = False
        self._result_queue = queue.Queue()
        self.on_notification: Optional[Callable[[str, Dict[str, Any]], None]] = None
        self._target_state = "unknown"
        self._last_stream_output: List[str] = []

    def _find_binary(self, name: str) -> str:
        # 1. Check if in PATH
        path = shutil.which(name)
        if path: return path

        if os.name == 'nt': # Windows specific discovery
            name_ext = f"{name}.exe"
            user_profile = os.environ.get('USERPROFILE', '')
            program_files = os.environ.get('ProgramFiles', 'C:\\Program Files')
            
            # Common search patterns
            search_patterns = [
                os.path.join(user_profile, "Downloads", "**", name_ext),
                "C:\\ST\\**\\" + name_ext,
                os.path.join(program_files, "**", name_ext),
                os.path.join(user_profile, "AppData", "Local", "**", name_ext)
            ]
            
            print(f"[Discovery] Searching for {name_ext} in common folders...")
            for pattern in search_patterns:
                matches = glob.glob(pattern, recursive=True)
                if matches:
                    found = matches[0]
                    print(f"[Discovery] Found {found}")
                    return found

        return name

    def connect(self, target_remote: str = "localhost:3333", elf_path: Optional[str] = None) -> bool:
        cmd = [self.binary_path, "--interpreter=mi2"]
        if elf_path:
            cmd.extend([elf_path])
        
        try:
            self.process = subprocess.Popen(
                cmd,
                stdin=subprocess.PIPE,
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
                text=True,
                bufsize=1
            )
            
            # Start background reader
            self._is_running = True
            self._reader_thread = threading.Thread(target=self._reader_loop, daemon=True)
            self._reader_thread.start()

            # 2. Connect to remote target
            res = self.send_command(f"target remote {target_remote}", timeout=10.0)
            if "^done" not in res:
                print(f"[GDB] Connection error: {res}")
                return False
            
            # 3. Breakthrough: Enable Hardware Vector Catch (Zero-Touch Debugging)
            # This forces the CPU to halt immediately on any fault at the EXACT instruction,
            # preserving CFSR and address registers before the system cascades to lockup.
            self.send_command("monitor arm semihosting enable")
            self.send_command("monitor reset halt")
            self.send_command("monitor arm vector_catch all")
            
            # Start the target running automatically
            self.send_command("continue")
            
            self._connected = True
            return True
        except Exception as e:
            print(f"[GDB] Startup failed: {e}")
            return False

    def disconnect(self) -> None:
        self._is_running = False
        if self.process:
            self.send_command("-gdb-exit")
            self.process.terminate()
            self.process = None
        self._connected = False

    def is_connected(self) -> bool:
        return self._connected and self.process is not None and self.process.poll() is None

    def get_state(self) -> str:
        """ Returns the current target state (e.g., 'halted', 'running'). """
        return self._target_state

    def send_command(self, cmd: str, timeout: float = 5.0) -> str:
        """ 
        Synchronously sends an MI command and waits for the result (^done, ^error, ^running).
        """
        if not self.process or not self.process.stdin:
            return ""

        # Flush any stale results
        while not self._result_queue.empty():
            self._result_queue.get()
        self._last_stream_output = []

        self.process.stdin.write(cmd + "\n")
        self.process.stdin.flush()

        try:
            # Wait for result record (starts with ^)
            res = self._result_queue.get(timeout=timeout)
            # Combine result with text stream output
            stream_out = "\n".join(self._last_stream_output)
            return f"{res}\n{stream_out}"
        except queue.Empty:
            return "^error,msg=\"Command timed out\""

    def read_registers(self, regs: List[str]) -> Dict[str, int]:
        results = {}
        for reg in regs:
            # -data-evaluate-expression is reliable for single registers
            res = self.send_command(f"-data-evaluate-expression ${reg}")
            match = re.search(r'value="([^"]+)"', res)
            if match:
                val_str = match.group(1).split()[0]
                try:
                    results[reg] = int(val_str, 16) if '0x' in val_str else int(val_str)
                except ValueError:
                    continue
        return results

    def read_memory(self, address: int, size: int) -> bytes:
        res = self.send_command(f"-data-read-memory-bytes {hex(address)} {size}")
        match = re.search(r'contents="([^"]+)"', res)
        if match:
            return bytes.fromhex(match.group(1))
        return b""

    def read_scb_fault_registers(self) -> Dict[str, int]:
        """
        Precision capture of SCB registers using absolute addresses.
        Offsets from 0xE000ED28: CFSR(0), HFSR(4), DFSR(8), MMFAR(12), BFAR(16).
        """
        results = {}
        # CFSR is a 3-in-1 register (MMFSR, BFSR, UFSR)
        results["cfsr"]  = self.read_mem(0xE000ED28, 4)
        results["hfsr"]  = self.read_mem(0xE000ED2C, 4)
        # We also read the Address registers directly to avoid relative offset errors
        results["mmfar"] = self.read_mem(0xE000ED34, 4)
        results["bfar"]  = self.read_mem(0xE000ED38, 4)
        return results

    def get_reg(self, name: str) -> Optional[int]:
        """Convenience method to get a single register value."""
        regs = self.read_registers([name])
        return regs.get(name)

    def execute(self, cmd: str) -> str:
        """Alias for send_command to match user expectation."""
        return self.send_command(cmd)

    def read_mem(self, address: Any, size: int) -> int:
        """Read 4 bytes from memory and return as integer (standard for registers)."""
        # Convert hex string address if needed
        if isinstance(address, str) and address.startswith("0x"):
            address = int(address, 16)
        
        raw = self.read_memory(address, size)
        if len(raw) >= size:
            import struct
            if size == 4:
                return struct.unpack("<I", raw)[0]
            elif size == 1:
                return struct.unpack("B", raw)[0]
        return 0

    def _reader_loop(self):
        """ 
        Continuously reads GDB output and classifies records.
        Records: Result (^), Notification (*), Stream (~, @, &)
        """
        while self._is_running and self.process and self.process.stdout:
            line = self.process.stdout.readline()
            if not line:
                break
            
            line = line.strip()
            if not line or line == "(gdb)":
                continue

            # Classify record
            if line.startswith('^'):
                # Result Record (Response to a command)
                self._result_queue.put(line)
            elif line.startswith('*'):
                # Async Notification (Event happened in target)
                self._handle_async_notification(line)
            elif line.startswith('~'):
                # Console Stream (Text output from commands)
                # Strip leading ~ and quotes
                match = re.search(r'~"(.+)"', line)
                if match:
                    # Unescape \n, \t, etc.
                    text = match.group(1).encode('utf-8').decode('unicode_escape')
                    self._last_stream_output.append(text.strip())
            # Stream records (@, &) are ignored or logged in debug

    def _handle_async_notification(self, line: str):
        """ Parses notifications like *stopped,reason="signal-received" """
        if line.startswith("*stopped"):
            # Extract reason
            reason_match = re.search(r'reason="([^"]+)"', line)
            reason = reason_match.group(1) if reason_match else "unknown"
            
            # Simple attribute parser for MI
            attrs = {}
            for pair in re.finditer(r'([a-zA-Z0-9_-]+)="([^"]+)"', line):
                attrs[pair.group(1)] = pair.group(2)
            
            if self.on_notification:
                self.on_notification("stopped", attrs)
            
            # Update internal state
            self._target_state = "halted"
        elif line.startswith("*running"):
            self._target_state = "running"

