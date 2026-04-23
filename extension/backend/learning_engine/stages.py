"""
Learning Engine - Stages & Workflow State Machine
"""

import enum
from typing import Dict, Any, Optional
from dataclasses import dataclass, field

class LearningStage(enum.Enum):
    INTENT_DISCOVERY = "intent_discovery"
    HARDWARE_TEACHING = "hardware_teaching"
    HARDWARE_VALIDATION = "hardware_validation"
    FIRMWARE_GENERATION = "firmware_generation"
    EXPLANATION = "explanation"

@dataclass
class LearningSession:
    """Tracks the state of a user's learning session."""
    session_id: str
    stage: LearningStage = LearningStage.INTENT_DISCOVERY
    user_intent: Optional[str] = None
    proposed_hardware: Dict[str, Any] = field(default_factory=dict)
    hardware_approved: bool = False
    generated_firmware: Optional[Dict[str, Any]] = None
    
    def advance(self, next_stage: LearningStage):
        """Transition to next stage."""
        print(f"[Session {self.session_id}] Transition: {self.stage.value} -> {next_stage.value}")
        self.stage = next_stage

class StageManager:
    """Manages transitions and state for learning sessions."""
    
    def __init__(self):
        self.sessions: Dict[str, LearningSession] = {}
        
    def get_session(self, session_id: str) -> LearningSession:
        if session_id not in self.sessions:
            self.sessions[session_id] = LearningSession(session_id=session_id)
        return self.sessions[session_id]
        
    def handle_intent(self, session_id: str, intent: str):
        """User states what they want to build."""
        session = self.get_session(session_id)
        session.user_intent = intent
        session.advance(LearningStage.HARDWARE_TEACHING)
        return session
        
    def handle_hardware_proposal(self, session_id: str, hardware: Dict[str, Any]):
        """AI proposes hardware."""
        session = self.get_session(session_id)
        session.proposed_hardware = hardware
        # Stay in Teaching until approved
        
    def approve_hardware(self, session_id: str):
        """User confirms hardware."""
        session = self.get_session(session_id)
        session.hardware_approved = True
        session.advance(LearningStage.FIRMWARE_GENERATION)
        return session
