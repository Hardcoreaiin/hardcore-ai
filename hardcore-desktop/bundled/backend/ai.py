"""
Gemini-Only Firmware Generation Engine with Intent Protection
Enhanced version of strict Gemini engine with semantic validation.
"""

import os
import json
import re
import time
import requests
from typing import Dict, Any, List, Optional, Tuple
from dataclasses import dataclass, field
from pathlib import Path
from datetime import datetime

from context_loader import ProjectContext, ProjectContextLoader

@dataclass
class GenerationAttempt:
    """Record of a single generation attempt."""
    model: str
    status_code: int
    latency_ms: int
    error_snippet: Optional[str] = None
    success: bool = False

@dataclass
class FirmwarePackage:
    """Complete firmware package with all required sections - supports multi-file architecture."""
    # Multi-file architecture (new)
    files: Dict[str, str] = field(default_factory=dict)  # {"main.cpp": "...", "pins.h": "...", etc.}
    file_tree: str = ""  # Human-readable file tree preview
    
    # Legacy single-file (backward compatibility)
    _code: str = ""  # Internal storage for legacy code
    
    pin_block: str = ""
    pin_json: Dict[str, Any] = field(default_factory=dict)
    timeline: str = ""
    tests: List[str] = field(default_factory=list)
    
    # Metadata
    model_used: str = ""
    confidence: float = 1.0
    context_used: bool = False
    attempts: List[GenerationAttempt] = field(default_factory=list)
    
    @property
    def code(self) -> str:
        """Backward compatible: returns main.cpp from files dict, or legacy _code."""
        if self.files:
            # Return main.cpp or first .cpp file
            if "main.cpp" in self.files:
                return self.files["main.cpp"]
            for name, content in self.files.items():
                if name.endswith(".cpp") or name.endswith(".c"):
                    return content
        return self._code
    
    @code.setter
    def code(self, value: str):
        """Set legacy code field."""
        self._code = value
        # Also populate files if empty
        if not self.files and value:
            self.files["main.cpp"] = value
    
    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary for API response."""
        return {
            "code": self.code,  # Backward compatible
            "files": self.files,  # New multi-file
            "file_tree": self.file_tree,
            "pin_block": self.pin_block,
            "pin_json": self.pin_json,
            "timeline": self.timeline,
            "tests": self.tests,
            "metadata": {
                "model": self.model_used,
                "confidence": self.confidence,
                "context_aware": self.context_used,
                "multi_file": len(self.files) > 1,
                "attempts": [
                    {
                        "model": a.model,
                        "status": a.status_code,
                        "latency_ms": a.latency_ms,
                        "success": a.success
                    } for a in self.attempts
                ]
            }
        }


class GeminiError(Exception):
    """Base exception for Gemini-related errors."""
    def __init__(self, error_code: str, message: str, attempts: List[GenerationAttempt]):
        self.error_code = error_code
        self.message = message
        self.attempts = attempts
        super().__init__(message)

class StrictGeminiEngine:
    """
    Strict Gemini-only firmware generation engine with semantic intent protection.
    
    CRITICAL: NO LOCAL FALLBACKS - Fails explicitly when Gemini unavailable.
    """
    
    # Whitelisted Gemini models
    DEFAULT_MODEL_WHITELIST = [
        "gemini-3-flash-preview",
        "gemini-3-flash-lite-preview", 
        "gemini-2.0-flash",
        "gemini-2.0-flash-lite"
    ]
    
    # Retry configuration (Survival Mode for free tier)
    RETRY_CONFIG = {
        "max_attempts": int(os.getenv("GEMINI_MAX_RETRIES", "10")),
        "base_delay": float(os.getenv("GEMINI_RETRY_BASE_DELAY", "2.0")),
        "max_delay": 60.0,
        "backoff_factor": 1.5
    }
    
    def __init__(self, api_key: Optional[str] = None, workspace_root: Optional[Path] = None):
        self.api_key = api_key or os.getenv("LLM_API_KEY", "")
        
        # Load model whitelist from env or use default
        whitelist_str = os.getenv("GEMINI_MODEL_WHITELIST", "")
        if whitelist_str:
            self.model_whitelist = [m.strip() for m in whitelist_str.split(",")]
        else:
            self.model_whitelist = self.DEFAULT_MODEL_WHITELIST.copy()
        
        self.current_model_index = 0
        self.current_model = self.model_whitelist[0]
        
        # Context loader for project awareness
        self.context_loader = ProjectContextLoader(workspace_root)
        
        # Configuration
        self.timeout = 30
        self.temperature = 0.2
        
        # Track last user prompt for helpers
        self.last_user_prompt: Optional[str] = None
        
        print(f"[StrictGemini] ===== GEMINI-ONLY ENGINE (INTENT-PROTECTED) =====")
        print(f"[StrictGemini] Model whitelist: {self.model_whitelist}")
        print(f"[StrictGemini] Max retries: {self.RETRY_CONFIG['max_attempts']}")
        print(f"[StrictGemini] API Key configured: {'Yes' if self.api_key else 'NO - WILL FAIL'}")
    
    def preflight_check_models(self) -> Tuple[Optional[str], Optional[str]]:
        """
        Query Gemini API to find first supported model.
        Returns: (model_name, error_message)
        """
        print(f"[StrictGemini] Running preflight model check...")
        
        if not self.api_key:
            return (None, "No API key configured")
        
        endpoint = f"https://generativelanguage.googleapis.com/v1beta/models?key={self.api_key}"
        
        try:
            resp = requests.get(endpoint, timeout=10)
            resp.raise_for_status()
            models_data = resp.json()
            
            # Find first whitelisted model that supports generateContent
            for model_info in models_data.get("models", []):
                model_name = model_info.get("name", "").replace("models/", "")
                
                if model_name in self.model_whitelist:
                    methods = model_info.get("supportedGenerationMethods", [])
                    if "generateContent" in methods:
                        print(f"[StrictGemini] [OK] Found supported model: {model_name}")
                        return (model_name, None)
            
            available = [m.get("name", "").replace("models/", "") for m in models_data.get("models", [])]
            error_msg = f"No whitelisted models support generateContent. Available: {available[:5]}"
            print(f"[StrictGemini] [FAIL] Preflight failed: {error_msg}")
            return (None, error_msg)
            
        except requests.HTTPError as e:
            error_msg = f"Preflight HTTP error {e.response.status_code}: {e.response.text[:200]}"
            print(f"[StrictGemini] [FAIL] Preflight failed: {error_msg}")
            return (None, error_msg)
        except Exception as e:
            error_msg = f"Preflight check failed: {str(e)}"
            print(f"[StrictGemini] [FAIL] Preflight failed: {error_msg}")
            return (None, error_msg)
    
    def generate_firmware(self, 
                         user_request: str, 
                         board_type: str = "esp32",
                         project_id: str = "current_project") -> FirmwarePackage:
        """
        Main entry point - Strict Gemini-only generation with intent protection.
        
        Raises GeminiError if generation fails for any reason.
        NEVER returns local fallback code.
        """
        print(f"\n[StrictGemini] ===== FIRMWARE GENERATION REQUEST =====")
        print(f"[StrictGemini] Request: {user_request[:100]}...")
        print(f"[StrictGemini] Board (API): {board_type}")
        
        # Track for pin extraction
        self.last_user_prompt = user_request
        
        attempts: List[GenerationAttempt] = []
        
        # STAGE 1: Preflight check
        supported_model, error = self.preflight_check_models()
        if not supported_model:
            raise GeminiError(
                error_code="MODEL_NOT_SUPPORTED",
                message=f"No Gemini models available. {error}",
                attempts=attempts
            )
        
        # Use the preflight-validated model
        self.current_model = supported_model
        
        # STAGE 2: Load context
        print(f"\n[StrictGemini] STAGE 1: Loading project context...")
        context = self.context_loader.load(project_id)
        
        # CRITICAL: Extract board from user's prompt text FIRST
        # This prevents asking "What board?" when user says "Make this ESP32..."
        extracted_board = self._extract_board_from_prompt(user_request)
        if extracted_board:
            print(f"[StrictGemini]   Board extracted from prompt: '{extracted_board}'")
            context.board_type = extracted_board
        elif board_type and board_type.lower() not in ('unknown', 'none', ''):
            normalized = self._normalize_board_identifier(board_type)
            print(f"[StrictGemini]   Board from API: {board_type} -> {normalized}")
            context.board_type = normalized
        
        # If still no board, use default
        if not context.board_type or context.board_type.lower() in ('unknown', 'none', ''):
            context.board_type = "esp32dev"
            print(f"[StrictGemini]   Using default board: esp32dev")
        
        # STAGE 3: Build prompt with ABSOLUTE CORE RULE
        print(f"[StrictGemini] STAGE 2: Building prompt...")
        prompt = self._build_contextual_prompt(user_request, context)
        
        # STAGE 4: Generate with retry (exponential backoff)
        print(f"[StrictGemini] STAGE 3: Calling Gemini with retry...")
        raw_output, attempts = self._generate_with_retry(prompt)
        
        if not raw_output:
            # All retries exhausted
            raise GeminiError(
                error_code="LLM_UNAVAILABLE",
                message=f"All {len(attempts)} Gemini attempts failed. Check API status and try again in 60s.",
                attempts=attempts
            )
        
        # STAGE 5: Parse output
        print(f"[StrictGemini] STAGE 4: Parsing output...")
        try:
            package = self._parse_ai_output(raw_output, context.board_type)
            package.model_used = self.current_model
            package.context_used = context.existing_code is not None
            package.attempts = attempts
            
        except Exception as e:
            raise GeminiError(
                error_code="VALIDATION_FAILED",
                message=f"Gemini output parsing failed: {str(e)}. Output may be incomplete.",
                attempts=attempts
            )
        
        # STAGE 6: SEMANTIC VALIDATION (Warning Only - NOT Blocking)
        # This checks if Gemini violated intent rules but does NOT prevent output
        print(f"[StrictGemini] STAGE 5: Semantic validation (warning only)...")
        is_semantically_valid, semantic_error = self._semantic_validate(user_request, package)
        
        if not is_semantically_valid:
            print(f"[StrictGemini]   ⚠️  WARNING: {semantic_error}")
            print(f"[StrictGemini]   ⚠️  Gemini may have misunderstood the request")
            print(f"[StrictGemini]   ℹ️  Returning output anyway (no blocking)")
            # Continue anyway - show what Gemini actually generated
        else:
            print(f"[StrictGemini]   ✓ Semantic validation passed")
        
        # STAGE 7: Strict structural validation
        print(f"[StrictGemini] STAGE 6: Structural validation...")
        is_valid, validation_error = self._validate_strict(package)
        
        if not is_valid:
            raise GeminiError(
                error_code="VALIDATION_FAILED",
                message=f"Gemini output incomplete: {validation_error}. Please try again.",
                attempts=attempts
            )
        
        # Save to history
        self.context_loader.save_history(project_id, user_request, package.to_dict())
        
        print(f"[StrictGemini] ✅ Generation complete!")
        print(f"[StrictGemini]   Model: {package.model_used}")
        print(f"[StrictGemini]   Board: {context.board_type}")
        print(f"[StrictGemini]   Confidence: {package.confidence}")
        
        return package
    
    def _build_contextual_prompt(self, user_request: str, context: ProjectContext) -> str:
        """Build prompt with ABSOLUTE CORE RULE and context injection."""
        
        system_prompt = """You are HardcoreAI — a senior embedded systems engineer, firmware architect, and robotics co-pilot.

Your job: Make hardware programming effortless for beginners and precise for professionals.
The user should be able to plug in a board, describe behavior in plain English, and get correct, flashable firmware.

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
CORE BEHAVIOR RULES (ABSOLUTE)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

YOU ARE BEING CALLED TO GENERATE FIRMWARE.
The user has ALREADY confirmed they want code.
Intent classification is handled BEFORE this call.

YOUR ONLY JOB: Generate complete, working firmware code.

🚫 DO NOT ask clarifying questions
🚫 DO NOT output summaries asking "Proceed?"
🚫 DO NOT return just a few sentences
✅ DO generate COMPLETE firmware with all sections

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
HARDWARE IS SOURCE OF TRUTH
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

• If system detects a board (USB / CP210x / CH340 / CMSIS-DAP):
  → Treat it as ACTIVE TARGET BOARD
  → NEVER ask which board again
  → NEVER override or contradict detection

• If no board detected:
  → Ask ONCE: "Which board are you using?"
  → Accept family-level answers (ESP32, STM32, Arduino, RP2040)
  → Internally select safe default variant

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
SMART CLARIFICATION (NO LOOPS)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

• Ask ONLY ONE question at a time
• NEVER repeat the same question
• NEVER reset conversation context
• If user answers, MERGE the answer and continue
• Prefer safe assumptions over blocking questions

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
PIN HANDLING (CRITICAL)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

• If user provides pins (ENA-19, IN1-21):
  → Parse naturally, validate they exist, use them exactly

• If user doesn't know pins:
  → Auto-assign REAL, SAFE GPIOs for that board
  → Avoid boot-critical pins (GPIO0, GPIO2, EN, BOOT)
  → Clearly list chosen pins before coding

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
WHEN ASKED TO GENERATE CODE — DO IT DIRECTLY
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

The user has ALREADY confirmed they want code. Generate it now.

DO NOT ask "Proceed?" — the UI already handled confirmation.
DO NOT output a summary and wait — generate the actual code.
DO NOT return just a clarification — return COMPLETE firmware.

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
ABSOLUTE CORE RULE (NON-NEGOTIABLE)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

THE USER'S INTENT IS THE SINGLE SOURCE OF TRUTH.

Once a task is identified (motor, sensor, robot):
• NEVER change, downgrade, or replace the task
• Motors → LEDs ❌ FORBIDDEN
• Driver-based → GPIO demo ❌ FORBIDDEN
• Ignoring user pins ❌ FORBIDDEN

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
MULTI-FILE FIRMWARE ARCHITECTURE (MANDATORY)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Generate MODULAR, PROFESSIONAL firmware as separate files:

1. **FILE TREE** (show structure first)
   src/
     main.cpp
     pins.h
     <component>.h
     <component>.cpp

2. **EACH FILE** with clear header:
   // FILE: <filename>
   ```cpp
   ...code...
   ```

3. **PIN CONNECTIONS** (human readable)

4. **<!--PIN-CONNECTIONS-JSON-->**

5. **TIMELINE** (first ~30 seconds behavior)

6. **TEST CHECKLIST**

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
FILE RULES (STRICT)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

• **main.cpp**: Entry point ONLY. Calls component APIs. NO raw GPIO here.
• **pins.h**: ALL #define PIN_XX. Single source of truth. NO pin numbers in .cpp files.
• **<component>.h**: Function declarations (init, control, etc.)
• **<component>.cpp**: Implementation. Include pins.h for pin numbers.

NEVER dump all code in main.cpp.
NEVER duplicate pin definitions.
NEVER hardcode pins inside .cpp files.

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
TIMING RULES
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

• Multi-phase logic MUST NOT use delay()
• Arduino-class MCUs → millis()-based FSM
• ESP32 → FreeRTOS tasks/timers when appropriate
• Parse complex timing exactly ("first X sec, then Y sec, then stop")

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
COMMUNICATION PROTOCOLS (COMPREHENSIVE)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

When user needs networking/communication, generate a comm/ folder with:

**INDUSTRIAL PROTOCOLS:**
• Modbus RTU (UART) - PLC, industrial sensors, meters
• Modbus TCP (Ethernet) - Industrial ethernet
• PROFINET - Siemens industrial
• EtherCAT - High-speed industrial
• CANopen - Industrial CAN networks
• OPC UA - Industry 4.0 standard

**AUTOMOTIVE & VEHICLE:**
• CAN Bus - Vehicle networks, robots
• CAN-FD - High-speed CAN
• LIN Bus - Low-cost vehicle sensors
• J1939 - Heavy vehicles, trucks
• FlexRay - High-speed automotive

**IoT & WIRELESS:**
• MQTT - IoT cloud messaging
• CoAP - Constrained devices
• HTTP/REST - Web APIs
• WebSocket - Real-time web
• AMQP - Message queuing

**SHORT-RANGE WIRELESS:**
• BLE (Bluetooth Low Energy) - Mobile apps, wearables
• Bluetooth Classic - Audio, SPP
• Wi-Fi - TCP/UDP over WiFi
• Zigbee - Home automation, mesh
• Z-Wave - Smart home
• Thread - IoT mesh

**LONG-RANGE WIRELESS:**
• LoRa/LoRaWAN - Long range IoT
• Sigfox - LPWAN
• NB-IoT - Cellular IoT
• LTE-M - Cellular IoT

**BUILDING AUTOMATION:**
• BACnet - HVAC, building systems
• KNX - Smart building
• DALI - Lighting control
• DMX512 - Stage lighting

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
UNIVERSAL COMMUNICATION ENGINE
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

You are a UNIVERSAL COMMUNICATION EXPERT.
The user describes WHAT they want. YOU decide HOW it communicates.
NEVER ask for protocol names by default.

**CATEGORY-BASED REASONING (think in categories, not names):**
• Fieldbus (industrial) → Modbus, PROFINET, EtherCAT
• Message-based (cloud/IoT) → MQTT, CoAP, AMQP
• Frame-based (automotive/robotics) → CAN, CAN-FD, LIN
• Stream-based (serial/USB) → UART, USB
• Packet-based (Ethernet/IP) → TCP, UDP, HTTP
• Wireless low-power → BLE, Zigbee, LoRa, Thread
• Wireless high-bandwidth → WiFi, Cellular
• Time-sensitive networks → TSN, EtherCAT

**UNIVERSAL INTENT INFERENCE (AUTO-DECIDE):**
• "send data to cloud" / "dashboard" / "monitor" → MQTT or HTTP
• "talk to PLC" / "SCADA" / "factory" → Modbus RTU/TCP
• "vehicle" / "car" / "robot bus" / "motor controller" → CAN Bus
• "long range sensor" / "remote" / "outdoor" → LoRa
• "board-to-board" / "module" → UART/SPI/I2C
• "high-speed device" / "USB" → USB
• "factory network" / "Industry 4.0" → OPC-UA
• "wireless control" / "phone" → BLE or WiFi
• "OTA update" / "remote update" → HTTP or MQTT

**DECISION HIERARCHY:**
1. If user specifies protocol explicitly → Use it
2. If user describes function → Infer best protocol
3. If ambiguous → Choose safest industry default
4. NEVER block generation due to unknown protocol

**RULES (NON-NEGOTIABLE):**
✗ NEVER ask "what protocol do you want?"
✗ NEVER require domain knowledge from user
✗ NEVER generate placeholder protocol code
✓ ALWAYS infer from context
✓ ALWAYS document the chosen protocol and why
✓ ALWAYS generate production-grade code

**COMM FOLDER STRUCTURE:**
src/comm/
  ├── comm_manager.h/c  # Abstraction layer (ALWAYS include)
  ├── <protocol>.h/c    # Protocol implementation

**COMM_MANAGER API:**
comm_init(protocol);
comm_send(topic, data, len);
comm_receive(buffer, max_len);
comm_set_callback(handler);

**DEFAULTS (always document in output):**
• Modbus: 9600 baud, Slave ID 1
• MQTT: broker.example.com:1883
• CAN: 500kbps
• BLE: Auto-generate UUID
• LoRa: SF7, 125kHz, 868/915 MHz

━━━━━━━━━━━━━━━━━━━━━━━━━━━━
OUTPUT FORMAT (STRICT — NO EXCEPTIONS)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Output EXACTLY these 5 sections in order.
Fail to follow this and the system WILL reject your code.

SECTION 1: CODE
Fenced code block with language.
[CODE]
```cpp
// ... code ...
```
[/CODE]

SECTION 2: PIN CONNECTIONS
Human-readable wiring guide.
[PINS]
- Connect VCC to 3.3V
- Connect IN1 to GPIO 15
[/PINS]

SECTION 3: JSON CONNECTIONS
Strict JSON array.
[PIN_JSON]
{
  "connections": [
    {"pin": "15", "function": "MOTOR_IN1", "type": "GPIO", "connected_to": "Driver IN1"}
  ]
}
[/PIN_JSON]

SECTION 4: TIMELINE
Execution timeline (first 30s).
[TIMELINE]
0s: System init
1s: Motor start
[/TIMELINE]

SECTION 5: TESTS
User verification checklist.
[TESTS]
- [ ] Check power LED
- [ ] Verify motor spins
[/TESTS]
"""

        # Context injection
        prompt_parts = [system_prompt, "\n\n--- CURRENT CONTEXT ---\n"]
        prompt_parts.append(f"TARGET BOARD: {context.board_type.upper()}\n")
        
        # Existing code context
        if context.existing_code:
            prompt_parts.append(f"\nEXISTING CODE (for reference):\n```cpp\n{context.existing_code[:800]}\n```\n")
            
        # Existing pins
        if context.existing_pins:
            pin_str = ", ".join(f"{k}={v}" for k, v in list(context.existing_pins.items())[:5])
            prompt_parts.append(f"\nCURRENT PINS: {pin_str}\n")
        
        # Conversation history
        if context.conversation_history:
            recent = context.conversation_history[-3:]
            prompt_parts.append("\n--- RECENT CONVERSATION ---\n")
            for i, entry in enumerate(recent, 1):
                user_msg = entry.get('user_prompt', '')
                action = entry.get('action', 'unknown')
                prompt_parts.append(f"{i}. User: \"{user_msg}\" → Action: {action}\n")
        
        # User request
        prompt_parts.append(f"\n--- USER REQUEST ---\n{user_request}\n")
        
        # Final instruction
        prompt_parts.append("\nGenerate complete firmware with ALL 5 sections. Ensure the hardware matches the user's EXACT intent.")
        
        return "".join(prompt_parts)
    
    def _semantic_validate(self, prompt: str, package: FirmwarePackage) -> Tuple[bool, Optional[str]]:
        """
        Semantic validation: ensure generated firmware matches user's actual intent.
        
        This catches violations like:
        - User asks for motor control, gets LED code
        - User provides explicit pins that are ignored
        """
        text = prompt.lower()
        code = package.code.lower()
        pin_block = (package.pin_block or "").lower()

        # CRITICAL CHECK 1: Detect LED substitution (most common violation)
        has_led_code = any(pattern in code for pattern in [
            "led_pin", "led =", "blink", "digitalwrite"
        ])
        user_asked_for_led = "led" in text or "blink" in text
        
        # If code is LED-focused but user never mentioned LEDs, check if they asked for something else
        if has_led_code and not user_asked_for_led:
            wants_motor = any(k in text for k in ["motor", "robot", "move", "l298", "driver"])
            wants_sensor = any(k in text for k in ["sensor", "read", "ultrasonic", "temperature"])
            
            if wants_motor or wants_sensor:
                return False, f"CRITICAL VIOLATION: User requested {('motor control' if wants_motor else 'sensor reading')} but generated code is LED blink demo."

        # CHECK 2: Motor / L298N detection
        wants_motor = any(k in text for k in ["motor", "robot", "move forward", "l298", "l298n"])
        has_motor_code = any(k in code for k in ["motor", "l298", "l298n"]) or ("pwm" in code and wants_motor)

        if wants_motor and not has_motor_code:
            return False, "Generated code does not implement motor control requested by user."

        # CHECK 3: Explicit pin labels must be reflected
        explicit_pins = self._extract_pin_mappings_from_prompt(prompt)
        if explicit_pins:
            missing_labels = []
            for conn in explicit_pins:
                label = conn.get("component", "")
                if not label:
                    continue
                l_lower = label.lower()
                if l_lower not in code and l_lower not in pin_block:
                    missing_labels.append(label)

            if missing_labels:
                return False, f"Generated code ignored explicit pin labels: {', '.join(missing_labels)}."

        return True, None
    
    def _extract_pin_mappings_from_prompt(self, prompt: str) -> List[Dict[str, Any]]:
        """
        Parse explicit user pin mappings such as:
        - ENA - 19
        - IN1 - 21
        - IN2: 18
        """
        connections: List[Dict[str, Any]] = []
        if not prompt:
            return connections

        for line in prompt.splitlines():
            m = re.match(r"^\s*([A-Za-z][A-Za-z0-9_ ]*?)\s*[-:=]\s*(\d+)\s*$", line)
            if not m:
                continue
            label, num = m.group(1).strip(), int(m.group(2))
            if not label:
                continue

            connections.append({
                "component": label,
                "pins": [{
                    "mcu_pin": num,
                    "component_pin": label,
                    "type": "GPIO"
                }]
            })

        if connections:
            print(f"[StrictGemini]   Extracted {len(connections)} explicit pin mappings")
        return connections
    
    def _extract_board_from_prompt(self, prompt: str) -> Optional[str]:
        """
        Extract board/MCU name from user's natural language.
        
        Examples:
        - "Make this ESP32 move forward" → "esp32dev"
        - "Connect L298N to Arduino Uno" → "arduino_uno"
        """
        if not prompt:
            return None
            
        lower = prompt.lower()
        
        # Check for explicit board mentions (order matters - specific first)
        if "arduino uno" in lower or " uno " in lower:
            return "arduino_uno"
        if "arduino nano" in lower or " nano " in lower:
            return "arduino_nano"
        if "arduino mega" in lower or " mega" in lower:
            return "arduino_mega"
        if "esp32" in lower:
            return "esp32dev"
        if "esp8266" in lower:
            return "esp12e"
        if "stm32" in lower:
            return "stm32"
        if "c2000" in lower or "ti c2000" in lower:
            return "ti_c2000"
        if "pico" in lower or "rp2040" in lower:
            return "rp2040"
            
        return None
    
    def _normalize_board_identifier(self, raw: str) -> str:
        """Normalize board identifier from UI/API."""
        s = raw.strip().lower()

        if "arduino nano" in s or "nano" in s:
            return "arduino_nano"
        if "arduino uno" in s or "uno" in s:
            return "arduino_uno"
        if "arduino mega" in s or "mega" in s:
            return "arduino_mega"
        if "esp32" in s:
            return "esp32dev"
        if "esp8266" in s:
            return "esp12e"
        if "stm32" in s:
            return "stm32"
        if "c2000" in s or "ti" in s:
            return "ti_c2000"
        if "pico" in s:
            return "rp2040"

        return raw
    
    # ===== All original methods below (unchanged) =====
    
    def _generate_with_retry(self, prompt: str) -> Tuple[Optional[str], List[GenerationAttempt]]:
        """Call Gemini with exponential backoff retry."""
        attempts = []
        max_attempts = self.RETRY_CONFIG["max_attempts"]
        base_delay = self.RETRY_CONFIG["base_delay"]
        backoff_factor = self.RETRY_CONFIG["backoff_factor"]
        max_delay = self.RETRY_CONFIG["max_delay"]
        
        for attempt_num in range(max_attempts):
            start_time = time.time()
            
            try:
                print(f"[StrictGemini]   Attempt {attempt_num + 1}/{max_attempts}: {self.current_model}")
                
                endpoint = f"https://generativelanguage.googleapis.com/v1/models/{self.current_model}:generateContent"
                payload = {
                    "contents": [{"parts": [{"text": prompt}]}],
                    "generation_config": {
                        "temperature": self.temperature,
                        "top_p": 0.95,
                        "top_k": 40,
                        "max_output_tokens": 4096
                    }
                }
                
                response = requests.post(
                    f"{endpoint}?key={self.api_key}",
                    headers={"Content-Type": "application/json"},
                    json=payload,
                    timeout=self.timeout
                )
                
                latency_ms = int((time.time() - start_time) * 1000)
                
                attempt = GenerationAttempt(
                    model=self.current_model,
                    status_code=response.status_code,
                    latency_ms=latency_ms,
                    success=response.status_code == 200
                )
                
                print(f"[StrictGemini]   Status: {response.status_code} ({latency_ms}ms)")
                
                if response.status_code == 429:
                    attempt.error_snippet = response.text[:200]
                    attempts.append(attempt)
                    
                    if attempt_num < max_attempts - 1:
                        delay = min(base_delay * (backoff_factor ** attempt_num), max_delay)
                        print(f"[StrictGemini]   Rate limited, retrying in {delay}s...")
                        time.sleep(delay)
                        self._rotate_model()
                        continue
                    else:
                        return (None, attempts)
                
                if response.status_code != 200:
                    attempt.error_snippet = response.text[:200]
                    attempts.append(attempt)
                    
                    if attempt_num < max_attempts - 1:
                        self._rotate_model()
                        time.sleep(0.5)
                        continue
                    else:
                        return (None, attempts)
                
                body = response.json()
                text = self._extract_text(body)
                
                if text:
                    print(f"[StrictGemini]   ✓ Received {len(text)} chars")
                    attempts.append(attempt)
                    return (text, attempts)
                else:
                    attempt.error_snippet = "Empty response"
                    attempts.append(attempt)
                    
                    if attempt_num < max_attempts - 1:
                        self._rotate_model()
                        continue
                    else:
                        return (None, attempts)
                    
            except requests.Timeout:
                latency_ms = int((time.time() - start_time) * 1000)
                attempt = GenerationAttempt(
                    model=self.current_model,
                    status_code=408,
                    latency_ms=latency_ms,
                    error_snippet="Timeout",
                    success=False
                )
                attempts.append(attempt)
                
                if attempt_num < max_attempts - 1:
                    self._rotate_model()
                    continue
                    
            except Exception as e:
                latency_ms = int((time.time() - start_time) * 1000)
                attempt = GenerationAttempt(
                    model=self.current_model,
                    status_code=500,
                    latency_ms=latency_ms,
                    error_snippet=str(e)[:200],
                    success=False
                )
                attempts.append(attempt)
                
                if attempt_num < max_attempts - 1:
                    self._rotate_model()
                    continue
        
        return (None, attempts)
    
    def _rotate_model(self):
        """Switch to next model in whitelist."""
        old_model = self.current_model
        self.current_model_index = (self.current_model_index + 1) % len(self.model_whitelist)
        self.current_model = self.model_whitelist[self.current_model_index]
        print(f"[StrictGemini]   Rotated: {old_model} → {self.current_model}")
    
    def _extract_text(self, response_body: Dict[str, Any]) -> Optional[str]:
        """Safely extract text from Gemini response."""
        try:
            candidates = response_body.get("candidates", [])
            if not candidates:
                return None
            
            parts = candidates[0].get("content", {}).get("parts", [])
            if not parts:
                return None
            
            return "\n".join(p.get("text", "") for p in parts).strip()
            
        except Exception:
            return None
    
    def _parse_ai_output(self, text: str, board: str) -> FirmwarePackage:
        """Parse AI output - extracts multi-file architecture or falls back to single file."""
        package = FirmwarePackage()
        
        # Try to extract multi-file output first
        files, file_tree = self._extract_multi_files(text)
        
        if files:
            # Multi-file architecture detected
            package.files = files
            package.file_tree = file_tree
            print(f"[AI] Multi-file output: {list(files.keys())}")
        else:
            # Fallback to legacy single-file extraction
            code = self._extract_code(text)
            if code:
                package._code = code
                package.files["main.cpp"] = code
                print(f"[AI] Single-file output (legacy)")
        
        package.pin_block = self._extract_pin_block(text, board)
        package.pin_json = self._extract_pin_json(text, board)
        package.timeline = self._extract_timeline(text)
        package.tests = self._extract_tests(text)
        package.confidence = self._calculate_confidence(package)
        
        # Regenerate pin_block from JSON for frontend
        if package.pin_json and package.pin_json.get("connections"):
            package.pin_block = self.generate_pin_block_from_json(package.pin_json, board)
        
        return package
    
    def _extract_multi_files(self, text: str) -> Tuple[Dict[str, str], str]:
        """
        Extract multiple files from AI output.
        
        Looks for patterns like:
        // FILE: pins.h
        ```cpp
        ...code...
        ```
        
        Returns:
            (files_dict, file_tree_string)
        """
        files = {}
        file_tree = ""
        
        # Extract FILE TREE section
        tree_match = re.search(r'\*\*FILE TREE\*\*\s*\n((?:.*(?:src/|├|└|│|  ).*\n?)+)', text, re.IGNORECASE)
        if tree_match:
            file_tree = tree_match.group(1).strip()
        
        # Extract each file block
        # Pattern: // FILE: <filename> followed by code block
        file_pattern = r'//\s*FILE:\s*([^\n]+)\s*\n```(?:cpp|c|h)?\s*\n(.*?)```'
        matches = re.findall(file_pattern, text, re.DOTALL | re.IGNORECASE)
        
        for filename, code in matches:
            filename = filename.strip()
            code = code.strip()
            if filename and code:
                files[filename] = code
                print(f"[AI] Extracted file: {filename} ({len(code)} chars)")
        
        # If no explicit FILE: markers, try to extract by code block position
        if not files:
            # Look for multiple code blocks
            code_blocks = re.findall(r'```(?:cpp|c|h)?\s*\n(.*?)```', text, re.DOTALL)
            if len(code_blocks) >= 2:
                # Multiple code blocks detected - try to infer file names
                for i, block in enumerate(code_blocks):
                    block = block.strip()
                    if not block:
                        continue
                    
                    # Try to infer filename from content
                    filename = self._infer_filename_from_code(block, i)
                    files[filename] = block
        
        return files, file_tree
    
    def _infer_filename_from_code(self, code: str, index: int) -> str:
        """Infer filename from code content."""
        code_lower = code.lower()
        
        # Check for header guards (indicates .h file)
        if '#ifndef' in code_lower and '#define' in code_lower:
            guard_match = re.search(r'#ifndef\s+(\w+)_H', code, re.IGNORECASE)
            if guard_match:
                return f"{guard_match.group(1).lower()}.h"
        
        # Check for #include "pins.h" (indicates it uses pins.h)
        if '#include "pins.h"' in code_lower:
            # This is likely a .cpp file
            pass
        
        # Check for setup()/loop() - indicates main.cpp
        if 'void setup()' in code_lower and 'void loop()' in code_lower:
            return "main.cpp"
        
        # Check for only #defines (likely pins.h)
        lines = [l.strip() for l in code.split('\n') if l.strip() and not l.strip().startswith('//')]
        if all(l.startswith('#define') or l.startswith('#ifndef') or l.startswith('#endif') for l in lines[:5]):
            return "pins.h"
        
        # Default: use index
        filenames = ["main.cpp", "pins.h", "driver.h", "driver.cpp"]
        return filenames[index] if index < len(filenames) else f"file_{index}.cpp"

    
    def _validate_strict(self, package: FirmwarePackage) -> Tuple[bool, Optional[str]]:
        """Strict 5-section validation with auto-repair."""
        if not package.code or len(package.code.strip()) < 50:
            return (False, "CODE section missing or too short")
        
        if not package.pin_json or not isinstance(package.pin_json, dict):
            return (False, "PIN_JSON missing")
        
        if "connections" not in package.pin_json or not package.pin_json["connections"]:
            connections = self._extract_pins_from_code_to_json(package.code)
            if connections:
                package.pin_json["connections"] = connections
            else:
                simple_pins = self._find_any_pins_in_code(package.code)
                if simple_pins:
                    package.pin_json["connections"] = simple_pins
                else:
                    package.pin_json["connections"] = []
        
        if not package.timeline:
            package.timeline = self._generate_default_timeline(package.code)
        
        if not package.tests:
            package.tests = self._generate_default_tests(package.code)
        
        return (True, None)
    
    def _generate_default_timeline(self, code: str) -> str:
        """Generate basic timeline from code analysis."""
        delays = re.findall(r'delay\((\d+)\)', code)
        if delays:
            total_ms = sum(int(d) for d in delays[:5])
            return f"Executes with delays totaling ~{total_ms}ms in main loop"
        if 'millis()' in code:
            return "Time-based control using millis() for non-blocking execution"
        if 'servo' in code.lower():
            return "Servo control with initialization and position commands"
        return "Continuous execution in main loop"
    
    def _generate_default_tests(self, code: str) -> List[str]:
        """Generate basic test checklist."""
        tests = []
        if 'Serial.begin' in code:
            tests.append("Verify serial monitor shows output")
        if 'servo' in code.lower() or 'Servo' in code:
            tests.append("Verify servo movement")
        if 'LED' in code or 'digitalWrite' in code:
            tests.append("Verify LED behavior")
        if 'delay(' in code:
            delays = re.findall(r'delay\((\d+)\)', code)
            if delays:
                tests.append(f"Verify timing matches delays ({delays[0]}ms, etc.)")
        if not tests:
            tests.append("Upload code and verify expected behavior")
            tests.append("Check serial monitor for debug output")
        tests.append("Ensure no compilation errors")
        return tests
    
    def _extract_pins_from_code_to_json(self, code: str) -> List[Dict[str, Any]]:
        """Extract pin definitions from code."""
        connections = []
        seen_pins = set()
        
        for match in re.finditer(r'#define\s+(\w*(?:PIN|LED|SERVO|MOTOR)\w*)\s+(\d+)', code, re.I):
            name, pin = match.groups()
            if pin not in seen_pins:
                component = name.replace("_PIN", "").replace("PIN_", "")
                connections.append({
                    "component": component,
                    "pins": [{"mcu_pin": pin, "component_pin": "Signal", "type": "GPIO"}],
                    "notes": f"From #define {name}"
                })
                seen_pins.add(pin)
        
        for match in re.finditer(r'const\s+int\s+(\w*(?:PIN|LED|SERVO|MOTOR)\w*)\s*=\s*(\d+)', code, re.I):
            name, pin = match.groups()
            if pin not in seen_pins:
                component = name.replace("_PIN", "").replace("PIN_", "")
                connections.append({
                    "component": component,
                    "pins": [{"mcu_pin": pin, "component_pin": "Signal", "type": "GPIO"}],
                    "notes": f"From const {name}"
                })
                seen_pins.add(pin)
        
        return connections
    
    def _find_any_pins_in_code(self, code: str) -> List[Dict[str, Any]]:
        """Last resort: find ANY pin numbers."""
        connections = []
        seen_pins = set()
        
        for match in re.finditer(r'pinMode\s*\(\s*(\d+)\s*,', code):
            pin = match.group(1)
            if pin not in seen_pins:
                connections.append({
                    "component": f"Pin_{pin}",
                    "pins": [{"mcu_pin": pin, "component_pin": "GPIO", "type": "GPIO"}],
                    "notes": "From pinMode()"
                })
                seen_pins.add(pin)
        
        for match in re.finditer(r'digitalWrite\s*\(\s*(\d+)\s*,', code):
            pin = match.group(1)
            if pin not in seen_pins:
                connections.append({
                    "component": f"Pin_{pin}",
                    "pins": [{"mcu_pin": pin, "component_pin": "GPIO", "type": "GPIO"}],
                    "notes": "From digitalWrite()"
                })
                seen_pins.add(pin)
        
        for match in re.finditer(r'\.attach\s*\(\s*(\d+)\s*\)', code):
            pin = match.group(1)
            if pin not in seen_pins:
                connections.append({
                    "component": "Servo",
                    "pins": [{"mcu_pin": pin, "component_pin": "PWM", "type": "PWM"}],
                    "notes": "From servo.attach()"
                })
                seen_pins.add(pin)
        
        return connections
    
    def _extract_code(self, text: str) -> str:
        """Extract code from fenced blocks."""
        match = re.search(r"```(?:cpp|c|arduino|ino)\n([\s\S]*?)\n```", text, re.I)
        if match:
            return match.group(1).strip()
        
        match = re.search(r"```\n?([\s\S]*?)\n?```", text)
        if match:
            code = match.group(1).strip()
            if "#include" in code or "void setup" in code:
                return code
        
        return ""
    
    def _extract_pin_block(self, text: str, board: str) -> str:
        """Extract PIN CONNECTIONS block."""
        match = re.search(r"(PIN CONNECTIONS[\s\S]*?)(?:\n\n|<!--PIN-CONNECTIONS-JSON-->)", text, re.I)
        if match:
            return match.group(1).strip()
        return f"PIN CONNECTIONS\n---\nMCU: {board}\n(Auto-generated)"
    
    def generate_pin_block_from_json(self, pin_json: Dict[str, Any], board: str) -> str:
        """Generate human-readable PIN CONNECTIONS block."""
        if not pin_json or "connections" not in pin_json:
            return f"PIN CONNECTIONS\n---\nMCU: {board}\nNo connections"
        
        connections = pin_json.get("connections", [])
        if not connections:
            return f"PIN CONNECTIONS\n---\nMCU: {board}\nNo connections"
        
        lines = ["PIN CONNECTIONS", "=" * 50, f"MCU: {pin_json.get('mcu', board).upper()}", ""]
        
        for i, conn in enumerate(connections, 1):
            component = conn.get("component", "Unknown")
            lines.append(f"{i}. {component}")
            lines.append("   " + "-" * 40)
            
            for pin_info in conn.get("pins", []):
                mcu_pin = pin_info.get("mcu_pin", "?")
                comp_pin = pin_info.get("component_pin", "Signal")
                pin_type = pin_info.get("type", "GPIO")
                lines.append(f"   MCU Pin {mcu_pin} → {component} {comp_pin} ({pin_type})")
            
            notes = conn.get("notes", "")
            if notes:
                lines.append(f"   Note: {notes}")
            lines.append("")
        
        lines.append("=" * 50)
        lines.append(f"Total: {len(connections)} connections")
        
        return "\n".join(lines)
    
    def _extract_pin_json(self, text: str, board: str) -> Dict[str, Any]:
        """Extract PIN JSON with flexible matching."""
        label_pos = text.find("<!--PIN-CONNECTIONS-JSON-->")
        
        if label_pos != -1:
            search_text = text[label_pos + 27:]
            json_str = self._find_json_object(search_text)
            if json_str:
                try:
                    parsed = json.loads(json_str)
                    if isinstance(parsed, dict):
                        if "connections" not in parsed:
                            parsed["connections"] = []
                        if "mcu" not in parsed:
                            parsed["mcu"] = board
                        return parsed
                except:
                    pass
        
        # Find any JSON with connections/mcu
        start_idx = 0
        while True:
            start = text.find('{', start_idx)
            if start == -1:
                break
            
            json_str = self._find_json_object(text[start:])
            if json_str:
                try:
                    parsed = json.loads(json_str)
                    if isinstance(parsed, dict) and ("connections" in parsed or "mcu" in parsed):
                        if "mcu" not in parsed:
                            parsed["mcu"] = board
                        return parsed
                except:
                    pass
                start_idx = start + len(json_str)
            else:
                start_idx = start + 1
        
        # Extract from code
        code = self._extract_code(text)
        if code:
            connections = self._extract_pins_from_code_to_json(code)
            if connections:
                return {"mcu": board, "connections": connections, "auto_extracted": True}
        
        return {"mcu": board, "connections": []}
    
    def _find_json_object(self, text: str) -> Optional[str]:
        """Find first balanced JSON object."""
        start = text.find('{')
        if start == -1:
            return None
        
        depth = 0
        in_string = False
        escape = False
        
        for i in range(start, len(text)):
            ch = text[i]
            
            if escape:
                escape = False
                continue
            if ch == '\\':
                escape = True
                continue
            if ch == '"':
                in_string = not in_string
                continue
            if in_string:
                continue
            
            if ch == '{':
                depth += 1
            elif ch == '}':
                depth -= 1
                if depth == 0:
                    return text[start:i+1]
        
        return None
    
    def _extract_timeline(self, text: str) -> str:
        """Extract timeline."""
        match = re.search(r"//\s*TIMELINE[:\s]+(.*?)(?:\n|$)", text, re.I)
        if match:
            return match.group(1).strip()
        return ""
    
    def _extract_tests(self, text: str) -> List[str]:
        """Extract test checklist."""
        tests = []
        match = re.search(r"(?:TEST|CHECKLIST)[:\s]*([\s\S]*?)(?:\n\n|$)", text, re.I)
        if match:
            for line in match.group(1).split("\n"):
                line = line.strip()
                if line and (line.startswith("-") or line.startswith("•")):
                    tests.append(line.lstrip("-•").strip())
        return tests
    
    def _calculate_confidence(self, package: FirmwarePackage) -> float:
        """Calculate confidence score."""
        score = 0.0
        if package.code and len(package.code) > 100:
            score += 0.4
        if package.pin_json and package.pin_json.get("connections"):
            score += 0.3
        if package.timeline:
            score += 0.15
        if package.tests:
            score += 0.15
        return round(score, 2)
    
    def chat_response(self, prompt: str) -> str:
        """Handle conversational chat."""
        try:
            endpoint = f"https://generativelanguage.googleapis.com/v1/models/{self.current_model}:generateContent"
            payload = {
                "contents": [{"parts": [{"text": f"You are HardcoreAI. User: '{prompt}'. Reply in 2-3 sentences."}]}],
                "generation_config": {"temperature": 0.7, "max_output_tokens": 512}
            }
            
            response = requests.post(f"{endpoint}?key={self.api_key}",
                                   headers={"Content-Type": "application/json"},
                                   json=payload, timeout=10)
            
            if response.status_code == 429:
                return "I'm rate-limited. Try generating firmware!"
            
            if response.status_code == 200:
                text = self._extract_text(response.json())
                return text if text else "I'm having trouble responding."
            
            return "Error connecting to AI."
            
        except Exception as e:
            return f"Chat error: {str(e)[:50]}"


# Backward compatibility
ClaudeStyleFirmwareAI = StrictGeminiEngine
