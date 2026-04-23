import threading
import time
import re
from typing import Optional, Callable, Any

try:
    import serial
except ImportError:
    serial = None

class UARTAdapter:
    """
    Streams logs from hardware via serial port.
    Detects patterns like 'HardFault' to trigger analysis.
    """

    def __init__(self, port: str, baudrate: int = 115200):
        self.port = port
        self.baudrate = baudrate
        self.ser: Any = None
        self.running = False
        self.thread: Optional[threading.Thread] = None
        self.on_pattern_match: Optional[Callable[[str], None]] = None

    def connect(self) -> bool:
        if not serial:
            print("[UART] Error: 'pyserial' not installed.")
            return False
        
        try:
            self.ser = serial.Serial(self.port, self.baudrate, timeout=1)
            self.running = True
            self.thread = threading.Thread(target=self._listen_loop, daemon=True)
            self.thread.start()
            print(f"[UART] Connected to {self.port} at {self.baudrate}")
            return True
        except Exception as e:
            print(f"[UART] Connection failed: {e}")
            return False

    def disconnect(self) -> None:
        self.running = False
        if self.ser:
            self.ser.close()
            self.ser = None
        if self.thread:
            self.thread.join(timeout=2)

    def is_connected(self) -> bool:
        return self.ser is not None and self.ser.is_open

    def _listen_loop(self):
        buffer = ""
        while self.running and self.ser is not None:
            try:
                if self.ser.in_waiting > 0:
                    data = self.ser.read(self.ser.in_waiting).decode('utf-8', errors='ignore')
                    print(data, end='', flush=True) # Pipeline logs to stdout
                    
                    buffer += data
                    if "\n" in buffer:
                        lines = buffer.split("\n")
                        for line in lines[:-1]:
                            self._check_patterns(line)
                        buffer = lines[-1]
                else:
                    time.sleep(0.1)
            except Exception as e:
                print(f"[UART] Error: {e}")
                break

    def _check_patterns(self, line: str):
        # Professional pattern detection
        patterns = [
            r"HardFault",
            r"UsageFault",
            r"BusFault",
            r"MemManage"
        ]
        
        for p in patterns:
            if re.search(p, line, re.IGNORECASE):
                if self.on_pattern_match:
                    self.on_pattern_match(line)
                break
