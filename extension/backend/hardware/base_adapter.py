from abc import ABC, abstractmethod
from typing import Dict, Any, Optional, List

class HardwareAdapter(ABC):
    """
    Abstract base class for all hardware interaction tools.
    Provides a standardized interface for connection, state monitoring,
    and data extraction.
    """

    @abstractmethod
    def connect(self, **kwargs) -> bool:
        """ Establish connection to the hardware or tool server. """
        raise NotImplementedError()

    @abstractmethod
    def disconnect(self) -> None:
        """ Cleanly close the connection. """
        raise NotImplementedError()

    @abstractmethod
    def is_connected(self) -> bool:
        """ Check if the connection is currently alive. """
        raise NotImplementedError()

    @abstractmethod
    def get_state(self) -> str:
        """ Get target state: 'running', 'halted', 'unknown', 'fault'. """
        raise NotImplementedError()

    @abstractmethod
    def read_registers(self, regs: List[str]) -> Dict[str, int]:
        """ Read a list of register values from the target. """
        raise NotImplementedError()

    @abstractmethod
    def read_memory(self, address: int, size: int) -> bytes:
        """ Read a block of memory from the target. """
        raise NotImplementedError()
