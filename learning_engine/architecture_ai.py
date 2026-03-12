"""
HardcoreAI V2 Industrial Architecture Engine
Replaces the educational learning engine with a professional engineering architect.
"""

import os
import re
import json
import asyncio
from typing import Dict, Any, List, Optional, Callable
from dataclasses import dataclass, field

# Internal imports
import sys
current_file_dir = os.path.dirname(os.path.abspath(__file__))
orchestrator_dir = os.path.abspath(os.path.join(current_file_dir, "../Hardcore.ai/orchestrator"))
if orchestrator_dir not in sys.path:
    sys.path.append(orchestrator_dir)

from gemini_api import GeminiAPI

@dataclass
class ArchitectureReport:
    overview: str = ""
    hardware: str = ""
    memory_storage: str = ""
    security: str = ""
    firmware_pipeline: str = ""
    wireless: str = ""
    compliance: str = ""
    bom: str = ""
    risks: str = ""

    def to_dict(self) -> Dict[str, Any]:
        return {
            "overview": self.overview,
            "hardware": self.hardware,
            "memory_storage": self.memory_storage,
            "security": self.security,
            "firmware_pipeline": self.firmware_pipeline,
            "wireless": self.wireless,
            "compliance": self.compliance,
            "bom": self.bom,
            "risks": self.risks
        }

class ArchitectureAI:
    def __init__(self, api_key: Optional[str] = None):
        self.api_key = api_key or os.getenv("LLM_API_KEY", "")
        self.gemini = GeminiAPI(api_key=self.api_key)
        self.vendor_db_path = os.path.join(current_file_dir, "vendor_db.json")
        self.vendor_db = self._load_vendor_db()

    def _load_vendor_db(self) -> Dict[str, Any]:
        try:
            with open(self.vendor_db_path, 'r') as f:
                return json.load(f)
        except Exception as e:
            print(f"[ArchitectureAI] Warning: Failed to load vendor_db.json: {e}")
            return {}

    async def classify_intent(self, user_request: str, history: Optional[List[Dict[str, str]]] = None) -> str:
        """Determines if the user wants Architecture Design or Firmware Code."""
        user_lower = user_request.lower()
        
        # Immediate keyword priority for requests starting with "give the code" or "show me the code"
        if user_lower.startswith(("give the code", "show me the code", "write the code", "generate firmware")):
            return "code"
            
        history_context = ""
        if history:
            history_context = "\n".join([f"{m['role'].upper()}: {m['content']}" for m in history[-5:]])
            
        prompt = f"""
        CONVERSATION HISTORY:
        {history_context}
        
        NEW USER REQUEST: {user_request}
        
        Classify the NEW USER REQUEST into EXACTLY one of these two: 'architecture' or 'code'.
        
        RULES:
        1. If the user is asking for COMPONENT SELECTION, SECURITY ARCHITECTURE, or COMPLIANCE, respond 'architecture'.
        2. If the user is asking for FIRMWARE, C++, DRIVERS, or actual IMPLEMENTATION details, respond 'code'.
        3. If the user starts with "Give the code", "Make the code", or "Show code", ALWAYS respond 'code'.
        4. If the request is a MIX, prioritize 'code' if the user explicitly mentions code/firmware/software.
        
        Respond with ONLY 'architecture' or 'code'.
        """
        response = await self.gemini.generate_content_async(prompt)
        return response.strip().lower() if response else "architecture"

    async def _classify_processor(self, user_request: str, history: Optional[List[Dict[str, str]]] = None) -> str:
        """Classifies the processor as MCU or MPU based on the request and database."""
        user_lower = user_request.lower()
        
        mcu_examples = ["esp32", "stm32", "rp2040", "avr", "pic", "nrf52"]
        mpu_examples = ["i.MX", "sitara", "stm32mp1", "rz", "iot soc", "cortex-a"]
        
        if any(ex.lower() in user_lower for ex in mcu_examples):
            return "MCU"
        if any(ex.lower() in user_lower for ex in mpu_examples):
            return "MPU"
            
        prompt = f"""
        USER REQUEST: {user_request}
        
        Classify the target processor as 'MCU' or 'MPU'.
        MCU examples: ESP32, STM32 (Cortex-M), RP2040, nRF52.
        MPU examples: NXP i.MX series, TI Sitara, STM32MP1, Snapdragon IoT.
        
        Respond with ONLY 'MCU' or 'MPU'.
        """
        response = await self.gemini.generate_content_async(prompt)
        return response.strip().upper() if response else "MCU"

    async def generate_architecture(self, user_request: str, history: Optional[List[Dict[str, str]]] = None, callback: Optional[Callable[[str], None]] = None) -> ArchitectureReport:
        if callback: callback("Analyzing Industrial Requirements...")
        
        processor_type = await self._classify_processor(user_request, history)
        system_prompt = self._build_system_prompt(processor_type)
        
        history_context = ""
        if history:
            history_context = "\n".join([f"{m['role'].upper()}: {m['content']}" for m in history[-3:]])
            
        prompt = f"HISTORY:\n{history_context}\n\nUSER REQUEST: {user_request}\n\nDeliver the professional architecture report following the strict structure."
        
        raw_output = await self.gemini.generate_content_async(f"{system_prompt}\n\n{prompt}", status_callback=callback)
        
        if not raw_output:
             raise Exception("AI failed to respond. Check API status.")
             
        return self._parse_report_output(raw_output)

    async def generate_firmware(self, user_request: str, history: Optional[List[Dict[str, str]]] = None, architecture_context: Optional[str] = None, callback: Optional[Callable[[str], None]] = None) -> Dict[str, Any]:
        if callback: callback("Synthesizing Industrial Firmware...")
        
        processor_type = await self._classify_processor(user_request, history)
        
        # Professional Engineering Constraint: Don't generate firmware for MPU unless specifically asked for a co-processor
        if processor_type == "MPU" and not any(kw in user_request.lower() for kw in ["cortex-m", "m33", "m4", "rtos core", "firmware for core"]):
             return {
                 "firmware": "// MPU-class system detected (Application Processor).\n// Hardware Architecture involves Linux/QNX/Android stacks.\n// Firmware generation is suppressed. Refer to System Architecture Design for boot chain and BSP details.",
                 "files": {
                     "architecture_notice.md": "MPU systems utilize complex boot chains (ROM -> SPL -> U-Boot -> Kernel). HardcoreAI generates Architecture Designs for these systems. Use specific requests (e.g., 'firmware for Cortex-M33') if low-level code is needed."
                 },
                 "message": "MPU-class system detected. Architecture design prioritized over firmware."
             }

        history_context = ""
        if history:
            history_context = "\n".join([f"{m['role'].upper()}: {m['content']}" for m in history[-5:]])
            
        prompt = f"""
        You are the HardcoreAI V2 Firmware Architect. 
        Generate professional C++/C firmware based on the conversation context and user request.
        
        TARGET ARCHITECTURE TYPE: {processor_type}

        {f"ARCHITECTURE DESIGN CONTEXT: {architecture_context}" if architecture_context else ""}

        CONTEXT:
        {history_context}
        
        USER REQUEST: {user_request}
        
        STRICT RULES:
        1. Use industrial coding standards (MISRA-C style where applicable).
        2. Target the specific silicon discussed.
        3. Provide a complete main file and any necessary headers (config.h, etc.).
        4. NO PLACEHOLDERS. NO FAKE SECURITY CHECKS (e.g., bool checkSecureBoot() {{ return true; }}).
        5. If the user asks to "set up circuitry", generate hardware design guidance in comments if code isn't applicable.
        
        OUTPUT FORMAT: You MUST respond with a JSON object. No other text.
        {{
            "firmware": "FULL C++ CODE FOR main.cpp",
            "files": {{ 
                "main.cpp": "...", 
                "config.h": "...",
                "bom.md": "...",
                "circuitry_guidance.md": "..."
            }},
            "message": "Industrial firmware package successfully generated for targeted silicon."
        }}
        """
        raw_output = await self.gemini.generate_content_async(prompt, status_callback=callback)
        
        try:
            json_start = raw_output.find('{')
            json_end = raw_output.rfind('}') + 1
            if json_start != -1 and json_end > json_start:
                json_str = raw_output[json_start:json_end]
                return json.loads(json_str)
            
            # Fallback
            code_match = re.search(r'```(?:cpp|c\+\+|arduino)?\n(.*?)```', raw_output, re.DOTALL)
            if code_match:
                return {"firmware": code_match.group(1).strip(), "files": {"main.cpp": code_match.group(1).strip()}, "message": "Firmware extracted from markdown."}
                
            return {"firmware": raw_output, "files": {"main.cpp": raw_output}, "message": "Firmware generated as raw text."}
        except Exception as e:
            print(f"[ArchitectureAI] Firmware JSON Parse Error: {e}")
            return {"firmware": raw_output, "files": {"main.cpp": raw_output}, "message": "Firmware generated (Parse error)."}

    def _build_system_prompt(self, processor_type: str = "MCU") -> str:
        db_context = json.dumps(self.vendor_db, indent=2)
        
        mpu_requirements = """
        For MPU-class systems, focus on:
        - Boot Chain: ROM -> SPL -> U-Boot -> FIT Image -> Kernel
        - Secure Boot: HAB verification, EdgeLock Secure Enclave, signed FIT images.
        - Power Architecture: PMIC selection, power rails sequencing.
        - Memory Architecture: DDR4/LPDDR4 interface, termination, routing constraints.
        - Storage: eMMC 5.1, UHS-I SDIO.
        - Wireless Validation: Flag WiFi 7 as bleeding edge; suggest u-blox MAYA-W2 (IW612) or Murata Type 2AE.
        - EU RED Compliance: List EN 300 328, EN 301 893, EN 62368-1, EN 55032.
        - OTA: A/B partition layout, signed images, rollback protection.
        """ if processor_type == "MPU" else """
        For MCU-class systems, focus on:
        - Firmware architecture and driver setup.
        - Pin connections and peripheral configuration.
        - Bare-metal or RTOS (FreeRTOS/Zephyr) implementation.
        - Power Consumption optimization.
        """

        return f"""
        You are the HardcoreAI V2 Industrial Embedded Systems Architecture Copilot.
        Your goal is to provide professional-grade architecture designs for engineers and hardware startups.
        Target System Type: {processor_type}

        STRICT RULES:
        1. Tone: Professional engineering only (e.g., use "impedance matching", "signal integrity", "propagation delay"). No beginners tutorials.
        2. Validation: You MUST validate manufacturer accuracy against the provided VENDOR DATABASE.
        3. Output Architecture:
        {mpu_requirements}
        4. Sections: Every response MUST contain exactly these 9 sections using [SECTION_NAME] tags:
           - System Overview
           - Hardware Architecture (Include Block Diagram logic)
           - Memory & Storage Architecture (Include wiring/routing considerations)
           - Security Architecture (ROM Boot, Trusted Execution, Secure Enclaves - NO placeholder C code)
           - Firmware & Update Pipeline (A/B updates, firmware signing)
           - Wireless Architecture (Module selection, Antenna design)
           - Compliance Requirements (EU RED, FCC, CE standards)
           - High-Level BOM (Professional components)
           - Engineering Risks (Thermal, EMI, supply chain)
        5. Wireless Constraint: If WiFi 7 is requested, explain current industrial availability and suggest certified WiFi 6/6E modules instead.

        VENDOR DATABASE CONTEXT:
        {db_context}
        """

    def _parse_report_output(self, text: str) -> ArchitectureReport:
        report = ArchitectureReport()
        sections = {
            "System Overview": "overview",
            "Hardware Architecture": "hardware",
            "Memory & Storage Architecture": "memory_storage",
            "Security Architecture": "security",
            "Firmware & Update Pipeline": "firmware_pipeline",
            "Wireless Architecture": "wireless",
            "Compliance Requirements": "compliance",
            "High-Level BOM": "bom",
            "Engineering Risks": "risks"
        }

        for tag, attr in sections.items():
            pattern = rf'\[{tag}\](.*?)((?=\[)|$)'
            match = re.search(pattern, text, re.DOTALL | re.IGNORECASE)
            if match:
                setattr(report, attr, match.group(1).strip())
        
        return report

async def test():
    ai = ArchitectureAI()
    req = "I would like to build an embedded system using an i.MX93 with 16GB memory and 64GB flash, secure boot, signed firmware updates, WiFi 7, BLE 5.0, and EU RED compliance."
    report = await ai.generate_architecture(req, callback=print)
    print(json.dumps(report.to_dict(), indent=2))

if __name__ == "__main__":
    asyncio.run(test())
