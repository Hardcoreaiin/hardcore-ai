"""
ESP32 Deterministic Pin Analyzer — v2
=======================================
Improvements in v2:
- Variable name tracking: associates variable names with pin numbers
  so component inference uses the ACTUAL variable name (e.g. "ledPin" → LED)
- Scoped context: only searches the surrounding 5 lines for context keywords
- UART/SPI/I2C pins get correct component labels (not inherited from nearby vars)
- All original functionality preserved
"""

import re
import json
import os
from typing import Dict, List, Any, Optional, Tuple

# Load the pinout knowledge base once at module level
_PINOUT_PATH = os.path.join(os.path.dirname(__file__), "esp32_pinout.json")
with open(_PINOUT_PATH, "r", encoding="utf-8") as f:
    PINOUT_DB = json.load(f)

PIN_DATA: Dict[str, Any] = PINOUT_DB["pins"]
PROTOCOLS: Dict[str, Any] = PINOUT_DB["protocols"]
FUNC_MAP: Dict[str, Any] = PINOUT_DB["arduino_function_map"]


# ─────────────────────────────────────────────────────────────────────────────
# COMPONENT INFERENCE — keyword → component name
# ─────────────────────────────────────────────────────────────────────────────

COMPONENT_KEYWORDS: List[Tuple[str, str]] = [
    # Output devices
    ("led",          "LED"),
    ("light",        "LED"),
    ("blink",        "LED"),
    ("illuminate",   "LED"),
    ("rgb",          "RGB LED"),
    ("neopixel",     "RGB LED"),
    ("ws2812",       "RGB LED"),
    ("strip",        "RGB LED"),
    ("buzz",         "Buzzer"),
    ("beep",         "Buzzer"),
    ("tone",         "Buzzer"),
    ("alarm",        "Buzzer"),
    ("servo",        "Servo"),
    ("srv",          "Servo"),
    ("relay",        "Relay"),
    ("motor",        "DC Motor"),
    ("l298",         "DC Motor"),
    ("l293",         "DC Motor"),
    ("lcd",          "LCD Display"),
    ("oled",         "OLED Display"),
    ("ssd1306",      "OLED Display"),
    ("display",      "Display"),
    ("segment",      "7-Segment Display"),
    ("sevseg",       "7-Segment Display"),
    # Input devices
    ("button",       "Button"),
    ("btn",          "Button"),
    ("switch",       "Button"),
    ("press",        "Button"),
    ("push",         "Button"),
    ("pir",          "PIR Motion Sensor"),
    ("motion",       "PIR Motion Sensor"),
    ("presence",     "PIR Motion Sensor"),
    ("ultrasonic",   "Ultrasonic Sensor"),
    ("hcsr04",       "Ultrasonic Sensor"),
    ("sonar",        "Ultrasonic Sensor"),
    ("trig",         "Ultrasonic Sensor"),
    ("echo",         "Ultrasonic Sensor"),
    ("dht",          "DHT Sensor"),
    ("temperature",  "Temperature Sensor"),
    ("humidity",     "DHT Sensor"),
    ("ldr",          "Light Sensor (LDR)"),
    ("photoresist",  "Light Sensor (LDR)"),
    ("pot",          "Potentiometer"),
    ("potentiometer","Potentiometer"),
    ("knob",         "Potentiometer"),
    ("joystick",     "Joystick"),
    ("encoder",      "Rotary Encoder"),
    ("thermistor",   "Thermistor"),
    ("mq",           "Gas Sensor"),
    ("gas",          "Gas Sensor"),
    ("smoke",        "Gas Sensor"),
    ("ir",           "IR Sensor"),
    ("infrared",     "IR Sensor"),
    ("touch",        "Touch Sensor"),
    # Communication
    ("sda",          "I2C Device"),
    ("scl",          "I2C Device"),
    ("mosi",         "SPI Device"),
    ("miso",         "SPI Device"),
    ("sck",          "SPI Device"),
    ("gps",          "GPS Module"),
    ("gsm",          "GSM Module"),
    ("bluetooth",    "Bluetooth Module"),
    ("rfid",         "RFID Reader"),
    ("sd",           "SD Card"),
]

# Protocol-specific component labels (override inference for protocol pins)
PROTOCOL_COMPONENT_LABELS = {
    "UART_TX":  "USB Serial (TX)",
    "UART_RX":  "USB Serial (RX)",
    "I2C_SDA":  "I2C Bus (SDA)",
    "I2C_SCL":  "I2C Bus (SCL)",
    "SPI_CLK":  "SPI Bus (CLK)",
    "SPI_MOSI": "SPI Bus (MOSI)",
    "SPI_MISO": "SPI Bus (MISO)",
    "SPI_SS":   "SPI Bus (SS)",
}


def _get_pin_info(gpio_num: int) -> Optional[Dict]:
    """Look up a GPIO number in the pinout database."""
    for key, pin in PIN_DATA.items():
        if pin.get("gpio") == gpio_num:
            return pin
    return None


def _infer_component_from_name(var_name: str) -> Optional[str]:
    """
    Infer component from a variable/constant name.
    This is the primary inference method — uses the actual variable name
    the developer chose (e.g. 'ledPin' → LED, 'buttonPin' → Button).
    """
    name_lower = var_name.lower()
    for keyword, component in COMPONENT_KEYWORDS:
        if keyword in name_lower:
            return component
    return None


def _infer_component_from_context(lines: List[str], line_num: int, window: int = 5) -> Optional[str]:
    """
    Infer component from surrounding code context (comments, variable names).
    Only searches within `window` lines of the current line to avoid false matches.
    """
    start = max(0, line_num - window)
    end = min(len(lines), line_num + window)
    context = " ".join(lines[start:end]).lower()

    for keyword, component in COMPONENT_KEYWORDS:
        if keyword in context:
            return component
    return None


# ─────────────────────────────────────────────────────────────────────────────
# REGEX PATTERNS
# ─────────────────────────────────────────────────────────────────────────────

PATTERNS = {
    "pinMode":      re.compile(r'pinMode\s*\(\s*(\w+)\s*,\s*(INPUT|OUTPUT|INPUT_PULLUP|INPUT_PULLDOWN)\s*\)', re.IGNORECASE),
    "digitalWrite": re.compile(r'digitalWrite\s*\(\s*(\w+)\s*,\s*(HIGH|LOW|1|0)\s*\)', re.IGNORECASE),
    "digitalRead":  re.compile(r'digitalRead\s*\(\s*(\w+)\s*\)', re.IGNORECASE),
    "analogRead":   re.compile(r'analogRead\s*\(\s*(\w+)\s*\)', re.IGNORECASE),
    "analogWrite":  re.compile(r'analogWrite\s*\(\s*(\w+)\s*,\s*(\w+)\s*\)', re.IGNORECASE),
    "ledcAttachPin":re.compile(r'ledcAttachPin\s*\(\s*(\w+)\s*,\s*(\w+)\s*\)', re.IGNORECASE),
    "dacWrite":     re.compile(r'dacWrite\s*\(\s*(\w+)\s*,\s*(\w+)\s*\)', re.IGNORECASE),
    "Wire.begin":   re.compile(r'Wire\.begin\s*\(\s*(?:(\w+)\s*,\s*(\w+))?\s*\)', re.IGNORECASE),
    "SPI.begin":    re.compile(r'SPI\.begin\s*\(\s*(?:(\w+)\s*,\s*(\w+)\s*,\s*(\w+)\s*,\s*(\w+))?\s*\)', re.IGNORECASE),
    "Serial.begin": re.compile(r'Serial\.begin\s*\(\s*(\w+)\s*\)', re.IGNORECASE),
    "Serial2.begin":re.compile(r'Serial2\.begin\s*\(\s*(\w+)(?:\s*,\s*\w+\s*,\s*(\w+)\s*,\s*(\w+))?\s*\)', re.IGNORECASE),
    "servo.attach": re.compile(r'(\w+)\.attach\s*\(\s*(\w+)\s*\)', re.IGNORECASE),
    "tone":         re.compile(r'tone\s*\(\s*(\w+)\s*,\s*(\w+)\s*\)', re.IGNORECASE),
    "touchRead":    re.compile(r'touchRead\s*\(\s*(\w+)\s*\)', re.IGNORECASE),
    # Constant extraction
    "define":       re.compile(r'#define\s+(\w+)\s+(\d+)', re.IGNORECASE),
    "const_int":    re.compile(r'(?:const\s+)?int\s+(\w+)\s*=\s*(\d+)\s*;', re.IGNORECASE),
    "const_byte":   re.compile(r'(?:const\s+)?(?:byte|uint8_t)\s+(\w+)\s*=\s*(\d+)\s*;', re.IGNORECASE),
}

ARDUINO_ALIASES = {
    "LED_BUILTIN": 2,
    "A0": 36, "A1": 39, "A2": 34, "A3": 35,
    "A4": 32, "A5": 33, "A6": 25, "A7": 26,
    "SDA": 21, "SCL": 22,
    "MOSI": 23, "MISO": 19, "SCK": 18, "SS": 5,
    "RX": 3, "TX": 1, "RX2": 16, "TX2": 17,
}


def _resolve_pin(token: str, constants: Dict[str, int]) -> Optional[int]:
    """Resolve a pin token to an integer GPIO number."""
    token = token.strip()
    if token.isdigit():
        return int(token)
    if token in ARDUINO_ALIASES:
        return ARDUINO_ALIASES[token]
    if token in constants:
        return constants[token]
    return None


# ─────────────────────────────────────────────────────────────────────────────
# MAIN ANALYSIS FUNCTION
# ─────────────────────────────────────────────────────────────────────────────

def analyze_code(code: str) -> Dict[str, Any]:
    """
    Main entry point. Parses C++ Arduino/ESP32 code and returns
    a complete, deterministic pin connection map.
    """
    lines = code.splitlines()

    # Step 1: Extract all constant definitions AND build a name→gpio map
    constants: Dict[str, int] = {}
    # Also track: gpio_num → list of variable names that refer to it
    gpio_var_names: Dict[int, List[str]] = {}

    for pattern_name in ("define", "const_int", "const_byte"):
        for m in PATTERNS[pattern_name].finditer(code):
            name, value = m.group(1), m.group(2)
            gpio_num = int(value)
            constants[name] = gpio_num
            # Only track if it's a plausible GPIO number (0-39)
            if 0 <= gpio_num <= 39:
                gpio_var_names.setdefault(gpio_num, []).append(name)

    # Also track Arduino aliases
    for alias, gpio_num in ARDUINO_ALIASES.items():
        if alias in code:
            gpio_var_names.setdefault(gpio_num, []).append(alias)

    # Step 2: Track pin usage
    pin_usage: Dict[int, Dict[str, Any]] = {}
    protocols_used: Dict[str, Any] = {}
    warnings: List[str] = []

    def register_pin(gpio: int, mode: str, func: str, line_num: int = 0, token: str = ""):
        if gpio is None:
            return
        pin_info = _get_pin_info(gpio)

        if gpio not in pin_usage:
            pin_usage[gpio] = {
                "gpio": gpio,
                "mode": mode,
                "functions_used": [],
                "component": None,
                "pin_data": pin_info,
                "warnings": [],
                "var_names": gpio_var_names.get(gpio, []),
            }

        if func not in pin_usage[gpio]["functions_used"]:
            pin_usage[gpio]["functions_used"].append(func)

        # Update mode if not already set (or upgrade specificity)
        if pin_usage[gpio]["mode"] == "UNKNOWN":
            pin_usage[gpio]["mode"] = mode

        # ── Component Inference (priority order) ──
        # 1. Protocol override (most specific)
        if mode in PROTOCOL_COMPONENT_LABELS and pin_usage[gpio]["component"] is None:
            pin_usage[gpio]["component"] = PROTOCOL_COMPONENT_LABELS[mode]

        # 2. Variable name inference (very reliable)
        elif pin_usage[gpio]["component"] is None or pin_usage[gpio]["component"] in (
            "LED / Output Device", "Button / Sensor", "Analog Sensor", "Unknown Component"
        ):
            # Try all variable names associated with this GPIO
            for var_name in gpio_var_names.get(gpio, []):
                inferred = _infer_component_from_name(var_name)
                if inferred:
                    pin_usage[gpio]["component"] = inferred
                    break

            # 3. Token name inference (the raw token used in the function call)
            if pin_usage[gpio]["component"] is None and token:
                inferred = _infer_component_from_name(token)
                if inferred:
                    pin_usage[gpio]["component"] = inferred

            # 4. Scoped context inference (last resort)
            if pin_usage[gpio]["component"] is None:
                inferred = _infer_component_from_context(lines, line_num)
                if inferred:
                    pin_usage[gpio]["component"] = inferred

            # 5. Mode-based fallback
            if pin_usage[gpio]["component"] is None:
                if "OUTPUT" in mode:
                    pin_usage[gpio]["component"] = "LED / Output Device"
                elif "INPUT" in mode:
                    pin_usage[gpio]["component"] = "Button / Sensor"
                elif "ANALOG" in mode:
                    pin_usage[gpio]["component"] = "Analog Sensor"
                else:
                    pin_usage[gpio]["component"] = "Unknown Component"

        # ── Warnings ──
        if pin_info:
            if "boot_warning" in pin_info:
                w = f"GPIO{gpio}: {pin_info['boot_warning']}"
                if w not in warnings:
                    warnings.append(w)
            if not pin_info.get("safe_for_output") and "OUTPUT" in mode:
                w = f"⚠ GPIO{gpio} ({pin_info['name']}) is NOT safe for output!"
                if w not in warnings:
                    warnings.append(w)
            if not pin_info.get("safe_for_input") and "INPUT" in mode:
                w = f"⚠ GPIO{gpio} ({pin_info['name']}) is NOT safe for input!"
                if w not in warnings:
                    warnings.append(w)

    # Step 3: Parse each function call with line number context
    def get_line_num(match: re.Match) -> int:
        return code[:match.start()].count('\n')

    # --- pinMode ---
    for m in PATTERNS["pinMode"].finditer(code):
        pin_token, mode = m.group(1), m.group(2).upper()
        gpio = _resolve_pin(pin_token, constants)
        if gpio is not None:
            register_pin(gpio, mode, "pinMode", get_line_num(m), pin_token)

    # --- digitalWrite ---
    for m in PATTERNS["digitalWrite"].finditer(code):
        pin_token = m.group(1)
        gpio = _resolve_pin(pin_token, constants)
        if gpio is not None:
            register_pin(gpio, "OUTPUT", "digitalWrite", get_line_num(m), pin_token)

    # --- digitalRead ---
    for m in PATTERNS["digitalRead"].finditer(code):
        pin_token = m.group(1)
        gpio = _resolve_pin(pin_token, constants)
        if gpio is not None:
            register_pin(gpio, "INPUT", "digitalRead", get_line_num(m), pin_token)

    # --- analogRead ---
    for m in PATTERNS["analogRead"].finditer(code):
        pin_token = m.group(1)
        gpio = _resolve_pin(pin_token, constants)
        if gpio is not None:
            register_pin(gpio, "ANALOG_INPUT", "analogRead", get_line_num(m), pin_token)
            adc2_pins = [p["gpio"] for p in PROTOCOLS["ADC2"].values() if isinstance(p, dict)]
            if gpio in adc2_pins and ("WiFi" in code or "wifi" in code.lower()):
                warnings.append(f"⚠ GPIO{gpio} is ADC2 — cannot be used while WiFi is active!")

    # --- analogWrite / ledcAttachPin ---
    for m in PATTERNS["analogWrite"].finditer(code):
        pin_token = m.group(1)
        gpio = _resolve_pin(pin_token, constants)
        if gpio is not None:
            register_pin(gpio, "PWM_OUTPUT", "analogWrite", get_line_num(m), pin_token)

    for m in PATTERNS["ledcAttachPin"].finditer(code):
        pin_token = m.group(1)
        gpio = _resolve_pin(pin_token, constants)
        if gpio is not None:
            register_pin(gpio, "PWM_OUTPUT", "ledcAttachPin", get_line_num(m), pin_token)

    # --- dacWrite ---
    for m in PATTERNS["dacWrite"].finditer(code):
        pin_token = m.group(1)
        gpio = _resolve_pin(pin_token, constants)
        if gpio is not None:
            register_pin(gpio, "DAC_OUTPUT", "dacWrite", get_line_num(m), pin_token)
            if gpio not in [25, 26]:
                warnings.append(f"⚠ GPIO{gpio}: dacWrite only works on GPIO25 and GPIO26!")

    # --- Wire.begin (I2C) ---
    for m in PATTERNS["Wire.begin"].finditer(code):
        sda_token, scl_token = m.group(1), m.group(2)
        sda = _resolve_pin(sda_token, constants) if sda_token else 21
        scl = _resolve_pin(scl_token, constants) if scl_token else 22
        protocols_used["I2C"] = {"SDA": sda, "SCL": scl}
        register_pin(sda, "I2C_SDA", "Wire.begin", get_line_num(m), "SDA")
        register_pin(scl, "I2C_SCL", "Wire.begin", get_line_num(m), "SCL")

    # --- SPI.begin ---
    for m in PATTERNS["SPI.begin"].finditer(code):
        sck  = _resolve_pin(m.group(1), constants) if m.group(1) else 18
        miso = _resolve_pin(m.group(2), constants) if m.group(2) else 19
        mosi = _resolve_pin(m.group(3), constants) if m.group(3) else 23
        ss   = _resolve_pin(m.group(4), constants) if m.group(4) else 5
        protocols_used["SPI"] = {"SCK": sck, "MISO": miso, "MOSI": mosi, "SS": ss}
        register_pin(sck,  "SPI_CLK",  "SPI.begin", get_line_num(m), "SCK")
        register_pin(miso, "SPI_MISO", "SPI.begin", get_line_num(m), "MISO")
        register_pin(mosi, "SPI_MOSI", "SPI.begin", get_line_num(m), "MOSI")
        register_pin(ss,   "SPI_SS",   "SPI.begin", get_line_num(m), "SS")

    # --- Serial.begin ---
    for m in PATTERNS["Serial.begin"].finditer(code):
        protocols_used["UART0"] = {"TX": 1, "RX": 3, "baud": m.group(1)}
        register_pin(1, "UART_TX", "Serial.begin", get_line_num(m), "TX0")
        register_pin(3, "UART_RX", "Serial.begin", get_line_num(m), "RX0")

    # --- Serial2.begin ---
    for m in PATTERNS["Serial2.begin"].finditer(code):
        rx = _resolve_pin(m.group(2), constants) if m.group(2) else 16
        tx = _resolve_pin(m.group(3), constants) if m.group(3) else 17
        protocols_used["UART2"] = {"TX": tx, "RX": rx, "baud": m.group(1)}
        register_pin(tx, "UART_TX", "Serial2.begin", get_line_num(m), "TX2")
        register_pin(rx, "UART_RX", "Serial2.begin", get_line_num(m), "RX2")

    # --- servo.attach ---
    for m in PATTERNS["servo.attach"].finditer(code):
        obj_name, pin_token = m.group(1), m.group(2)
        if any(kw in obj_name.lower() for kw in ["servo", "srv", "motor"]):
            gpio = _resolve_pin(pin_token, constants)
            if gpio is not None:
                register_pin(gpio, "PWM_OUTPUT", "servo.attach", get_line_num(m), obj_name)
                if gpio in pin_usage:
                    pin_usage[gpio]["component"] = "Servo Motor"

    # --- tone ---
    for m in PATTERNS["tone"].finditer(code):
        pin_token = m.group(1)
        gpio = _resolve_pin(pin_token, constants)
        if gpio is not None:
            register_pin(gpio, "OUTPUT", "tone", get_line_num(m), pin_token)
            if gpio in pin_usage:
                pin_usage[gpio]["component"] = "Buzzer"

    # --- touchRead ---
    for m in PATTERNS["touchRead"].finditer(code):
        pin_token = m.group(1)
        gpio = _resolve_pin(pin_token, constants)
        if gpio is not None:
            register_pin(gpio, "TOUCH_INPUT", "touchRead", get_line_num(m), pin_token)
            if gpio in pin_usage:
                pin_usage[gpio]["component"] = "Touch Sensor"

    # Step 4: Build connection list
    connections = _build_connections(pin_usage)

    # Step 5: Summary
    output_count = sum(1 for p in pin_usage.values() if "OUTPUT" in p["mode"])
    input_count  = sum(1 for p in pin_usage.values() if "INPUT" in p["mode"] and "ANALOG" not in p["mode"])
    analog_count = sum(1 for p in pin_usage.values() if "ANALOG" in p["mode"])
    summary_parts = []
    if output_count: summary_parts.append(f"{output_count} digital output(s)")
    if input_count:  summary_parts.append(f"{input_count} digital input(s)")
    if analog_count: summary_parts.append(f"{analog_count} analog input(s)")
    for proto in protocols_used:
        summary_parts.append(proto)
    summary = ", ".join(summary_parts) if summary_parts else "No GPIO usage detected"

    return {
        "board": "ESP32-WROOM-32",
        "pins": list(pin_usage.values()),
        "protocols": protocols_used,
        "connections": connections,
        "warnings": warnings,
        "summary": summary,
        "constants_found": constants,
    }


# ─────────────────────────────────────────────────────────────────────────────
# CONNECTION BUILDER
# ─────────────────────────────────────────────────────────────────────────────

COLOR_MAP = {
    "OUTPUT":        "#22c55e",
    "PWM_OUTPUT":    "#f59e0b",
    "INPUT":         "#3b82f6",
    "INPUT_PULLUP":  "#60a5fa",
    "INPUT_PULLDOWN":"#93c5fd",
    "ANALOG_INPUT":  "#a78bfa",
    "DAC_OUTPUT":    "#f97316",
    "I2C_SDA":       "#ec4899",
    "I2C_SCL":       "#ec4899",
    "SPI_CLK":       "#14b8a6",
    "SPI_MOSI":      "#14b8a6",
    "SPI_MISO":      "#14b8a6",
    "SPI_SS":        "#14b8a6",
    "UART_TX":       "#f43f5e",
    "UART_RX":       "#fb923c",
    "TOUCH_INPUT":   "#84cc16",
}

def _build_connections(pin_usage: Dict) -> List[Dict]:
    connections = []
    for gpio, info in pin_usage.items():
        component = info.get("component", "Unknown")
        mode = info.get("mode", "UNKNOWN")
        color = COLOR_MAP.get(mode, "#9ca3af")
        connections.append({
            "from": f"ESP32:GPIO{gpio}",
            "to": f"{component}:Pin",
            "gpio": gpio,
            "mode": mode,
            "color": color,
            "label": f"GPIO{gpio} → {component}",
            "functions": info.get("functions_used", []),
            "var_names": info.get("var_names", []),
        })
    return connections


# ─────────────────────────────────────────────────────────────────────────────
# WOKWI DIAGRAM GENERATOR
# ─────────────────────────────────────────────────────────────────────────────

COMPONENT_TO_WOKWI = {
    "LED":                  "wokwi-led",
    "LED / Output Device":  "wokwi-led",
    "RGB LED":              "wokwi-led",
    "Buzzer":               "wokwi-buzzer",
    "Button":               "wokwi-pushbutton",
    "Button / Sensor":      "wokwi-pushbutton",
    "Servo Motor":          "wokwi-servo",
    "Potentiometer":        "wokwi-potentiometer",
    "Analog Sensor":        "wokwi-potentiometer",
    "Touch Sensor":         "wokwi-pushbutton",
    "Relay":                "wokwi-relay",
}

COMPONENT_COLORS = {
    "LED":               "red",
    "LED / Output Device":"green",
    "RGB LED":           "blue",
    "Buzzer":            "black",
    "Button":            "green",
    "Button / Sensor":   "blue",
    "Servo Motor":       "gray",
    "Potentiometer":     "gray",
}

def generate_wokwi_diagram(analysis: Dict) -> Dict:
    parts = [{
        "type": "wokwi-esp32-devkit-v1",
        "id": "esp32",
        "top": 0, "left": 0,
        "attrs": {}
    }]
    connections = []

    for i, info in enumerate(analysis["pins"]):
        gpio = info.get("gpio")
        component = info.get("component", "Unknown")
        wokwi_type = COMPONENT_TO_WOKWI.get(component)
        if not wokwi_type:
            continue

        part_id = f"{component.lower().replace(' ', '_').replace('/', '_')}_{gpio}"
        color = COMPONENT_COLORS.get(component, "red")

        parts.append({
            "type": wokwi_type,
            "id": part_id,
            "top": i * 80,
            "left": 300,
            "attrs": {"color": color}
        })

        wcolor = COLOR_MAP.get(info.get("mode", "OUTPUT"), "#22c55e")
        connections.append([f"esp32:D{gpio}", f"{part_id}:A", wcolor, []])

    return {
        "version": 1,
        "author": "HardcoreAI Deterministic Parser v2",
        "editor": "wokwi",
        "parts": parts,
        "connections": connections,
    }


# ─────────────────────────────────────────────────────────────────────────────
# SELF-TEST
# ─────────────────────────────────────────────────────────────────────────────

if __name__ == "__main__":
    test_code = """
    #define LED_PIN 2
    #define BUTTON_PIN 4
    const int BUZZER = 15;
    const int potPin = 34;
    const int servoPin = 13;

    Servo myServo;

    void setup() {
        Serial.begin(115200);
        pinMode(LED_PIN, OUTPUT);
        pinMode(BUTTON_PIN, INPUT_PULLUP);
        pinMode(BUZZER, OUTPUT);
        myServo.attach(servoPin);
    }

    void loop() {
        int potVal = analogRead(potPin);
        if (digitalRead(BUTTON_PIN) == LOW) {
            digitalWrite(LED_PIN, HIGH);
            tone(BUZZER, 1000);
            myServo.write(map(potVal, 0, 4095, 0, 180));
        } else {
            digitalWrite(LED_PIN, LOW);
            noTone(BUZZER);
        }
        delay(10);
    }
    """

    result = analyze_code(test_code)
    print("=== PIN ANALYSIS v2 ===")
    for gpio, info in result["pins"].items():
        var_str = f" [vars: {', '.join(info['var_names'])}]" if info['var_names'] else ""
        print(f"  GPIO{gpio}: {info['component']} ({info['mode']}) — {info['functions_used']}{var_str}")
    print("\n=== WARNINGS ===")
    for w in result["warnings"]:
        print(f"  {w}")
    print(f"\n=== SUMMARY ===\n  {result['summary']}")
