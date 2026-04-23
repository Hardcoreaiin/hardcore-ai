from abc import ABC, abstractmethod
from typing import Dict, Any

class ArchitectureAdapter(ABC):
    """
    Base interface for architecture-specific register capture and fault analysis.
    """

    @abstractmethod
    def detect(self, gdb_adapter) -> bool:
        """Return True if this architecture matches the current target."""
        pass

    @abstractmethod
    def capture_registers(self, gdb_adapter) -> Dict[str, Any]:
        """Return raw fault data/registers from the target."""
        pass

    @abstractmethod
    def analyze(self, data: Dict[str, Any]) -> Dict[str, Any]:
        """Return interpreted fault info for the diagnostic engine."""
        pass
