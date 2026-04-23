"""
HardcoreAI Learning Engine - AI Core
Refactored for Single-Pass Generation.
Solves 429 Rate Limit errors by consolidating all tasks into 1 call.
"""

import os
import re
import time
import json
import asyncio
from typing import Dict, Any, List, Optional, Tuple, Callable
from dataclasses import dataclass, field

# Internal imports
import sys
current_file_dir = os.path.dirname(os.path.abspath(__file__))
orchestrator_dir = os.path.abspath(os.path.join(current_file_dir, "../Hardcore.ai/orchestrator"))
if orchestrator_dir not in sys.path:
    sys.path.append(orchestrator_dir)

from gemini_api import GeminiAPI

@dataclass
class GenerationAttempt:
    model: str
    success: bool = False

@dataclass
class FirmwarePackage:
    """Complete firmware package with all pedagogy sections embedded."""
    files: Dict[str, str] = field(default_factory=dict)
    pin_json: Dict[str, Any] = field(default_factory=dict)
    theory: Dict[str, Any] = field(default_factory=dict)
    diagram: Dict[str, Any] = field(default_factory=dict)
    simulation_logic: Dict[str, Any] = field(default_factory=dict)
    execution_flow: List[Dict[str, Any]] = field(default_factory=list)
    explanation: Dict[str, Any] = field(default_factory=dict)
    
    def to_dict(self) -> Dict[str, Any]:
        return {
            "files": self.files,
            "pin_json": self.pin_json,
            "theory": self.theory,
            "diagram": self.diagram,
            "simulation_logic": self.simulation_logic,
            "execution_flow": self.execution_flow,
            "explanation": self.explanation
        }

class LearningAI:
    """
    AI Engine optimized for RELIABILITY and QUOTA management.
    Performs all generation in a single Unified Pass.
    """
    
    def __init__(self, api_key: Optional[str] = None):
        self.api_key = api_key or os.getenv("LLM_API_KEY", "")
        self.gemini = GeminiAPI(api_key=self.api_key)

    async def generate_firmware(self, user_request: str, hardware: Dict[str, Any], callback: Optional[Callable[[str], None]] = None) -> FirmwarePackage:
        """
        Unified Generation Pass. 
        Requests Code, Theory, Diagram, and Logic in one single API call.
        """
        if callback: callback("Synthesizing Complete Project Architecture...")
        
        prompt = self._build_unified_prompt(user_request, hardware)
        raw_output = await self.gemini.generate_content_async(prompt, status_callback=callback)
        
        if not raw_output:
             raise Exception("AI failed to respond. Check API status.")
             
        return self._parse_unified_output(raw_output)

    async def generate_learning_content(self, user_request: str, callback: Optional[Callable[[str], None]] = None) -> Dict[str, Any]:
        """Simple turn-based teaching (Intent Discovery)."""
        prompt = f"Role: Hardware Mentor. User: {user_request}. Propose components and explain logic in JSON format."
        return await self._call_json_async(prompt, callback=callback)

    async def _call_json_async(self, prompt: str, callback: Optional[Callable[[str], None]] = None) -> Any:
        text = await self.gemini.generate_content_async(prompt + "\n\nRespond in pure JSON.", status_callback=callback)
        try:
            return json.loads(re.sub(r'```(?:json)?\s*|\s*```', '', text).strip())
        except:
            return {"explanation": text}

    def _build_unified_prompt(self, request: str, hardware: Dict[str, Any]) -> str:
        return f"""
        You are HardcoreAI. Generate a complete embedded project in ONE PASS.
        
        USER REQUEST: {request}
        HARDWARE: {json.dumps(hardware)}
        
        STANDARD HARDWARE MAP (Use these GPIOs if applicable):
        - LED: GPIO 2 (Internal) or GPIO 4 (External)
        - Button: GPIO 18, 19
        - L298N Motor Driver: ENA=27, IN1=26, IN2=25, ENB=33, IN3=32, IN4=14
        - Servo: GPIO 13, 15
        - Ultrasonic (HC-SR04): TRIG=5, ECHO=18
        - I2C (OLED/LCD): SDA=21, SCL=22
        - Analog Sensors (Pot): GPIO 34, 35
        
        OUTPUT FORMAT (Strictly use these tags):
        
        [CODE]
        // FILE: main.cpp
        ```cpp
        #include <Arduino.h>
        ...
        ```
        [/CODE]
        
        [PIN_JSON]
        {{ "components": [...], "connections": [...] }}
        [/PIN_JSON]
        
        [THEORY]
        {{ "concept": "...", "electrical": "...", "code": "...", "execution": "...", "applications": "..." }}
        [/THEORY]
        
        [DIAGRAM]
        {{ "version": 1, "author": "HardcoreAI", "editor": "wokwi", "parts": [...], "connections": [...] }}
        [/DIAGRAM]
        
        [SIMULATION]
        {{ "logic": "function body here..." }}
        [/SIMULATION]
        
        [FLOW]
        [ {{"step": 1, "action": "...", "pin": 5, "value": 1}}, ... ]
        [/FLOW]
        
        [EXPLANATION]
        {{ "concept": "...", "theory": "Fast overview", "next_step": "..." }}
        [/EXPLANATION]
        """

    def _parse_unified_output(self, text: str) -> FirmwarePackage:
        pkg = FirmwarePackage()
        
        def extract(tag: str) -> str:
            match = re.search(rf'\[{tag}\](.*?)\[/{tag}\]', text, re.DOTALL | re.IGNORECASE)
            return match.group(1).strip() if match else ""

        # Extract Code
        code_block = extract("CODE")
        file_matches = re.findall(r'//\s*FILE:\s*([^\n]+)\s*\n```(?:cpp|c|h)?\s*\n(.*?)```', code_block, re.DOTALL | re.IGNORECASE)
        for name, content in file_matches:
            pkg.files[name.strip()] = content.strip()
        if not pkg.files:
            single = re.search(r'```(?:cpp|c)?\s*\n(.*?)```', code_block, re.DOTALL | re.IGNORECASE)
            if single: pkg.files["main.cpp"] = single.group(1).strip()

        # Extract JSON blocks
        for tag, attr in [("PIN_JSON", "pin_json"), ("THEORY", "theory"), ("DIAGRAM", "diagram"), ("SIMULATION", "simulation_logic"), ("FLOW", "execution_flow"), ("EXPLANATION", "explanation")]:
            raw = extract(tag)
            if raw:
                try: 
                    data = json.loads(raw)
                    setattr(pkg, attr, data)
                except: pass
                
        return pkg
