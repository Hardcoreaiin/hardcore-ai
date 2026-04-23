import re
import queue
import threading
import subprocess

class GDBMIParser:
    """
    Lightweight GDB/MI stream parser.
    Separates results, notifications, and stream output.
    """
    def __init__(self, stdout):
        self.stdout = stdout
        self.result_queue = queue.Queue()
        self.async_notifications = queue.Queue()
        self.streaming = True
        self.thread = threading.Thread(target=self._reader, daemon=True)
        self.thread.start()

    def _reader(self):
        while self.streaming:
            line = self.stdout.readline()
            if not line:
                break
            line = line.strip()
            if not line:
                continue

            # Identify the type of MI record
            if line.startswith('^'): # Result Record
                self.result_queue.put(line)
            elif line.startswith('*'): # Async Notification
                self.async_notifications.put(line)
            elif line.startswith('~') or line.startswith('@') or line.startswith('&'):
                # Console, target, or log stream
                pass
            elif "(gdb)" in line:
                # Prompt - signal completion for some commands
                pass

    def stop(self):
        self.streaming = False

# Testing the logic
def test_parse_halt_reason(line):
    # example: *stopped,reason="signal-received",signal-name="SIGTRAP",...
    match = re.search(r'reason="([^"]+)"', line)
    return match.group(1) if match else "unknown"

line = '*stopped,reason="signal-received",signal-name="SIGTRAP",signal-meaning="Trace/breakpoint trap",frame={addr="0x080001f0",func="main"}'
print(f"Halt Reason: {test_parse_halt_reason(line)}")
