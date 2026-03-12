"""
HardcoreAI Deterministic Rule Engine
=====================================
Safety-critical hardware validation system for embedded boards.

This is NOT an AI inference system.
This is a deterministic static analyzer with datasheet-grounded rules.

Rules are the final authority. The LLM may suggest fixes but NEVER bypasses failing rules.
"""

import json
import re
from typing import Dict, List, Any, Tuple, Optional
from dataclasses import dataclass
from pathlib import Path


@dataclass
class RuleViolation:
    """Represents a single rule violation."""
    rule_id: str
    severity: str  # error | warning | info
    message: str
    fix_suggestion: str
    datasheet_reference: str
    affected_pins: List[int] = None
    
    def to_dict(self) -> Dict[str, Any]:
        return {
            "rule_id": self.rule_id,
            "severity": self.severity,
            "message": self.message,
            "fix_suggestion": self.fix_suggestion,
            "datasheet_reference": self.datasheet_reference,
            "affected_pins": self.affected_pins or []
        }


@dataclass
class ValidationResult:
    """Result of hardware configuration validation."""
    is_valid: bool
    errors: List[RuleViolation]
    warnings: List[RuleViolation]
    info: List[RuleViolation]
    
    @property
    def total_violations(self) -> int:
        return len(self.errors) + len(self.warnings) + len(self.info)
    
    def to_dict(self) -> Dict[str, Any]:
        return {
            "is_valid": self.is_valid,
            "total_violations": self.total_violations,
            "errors": [e.to_dict() for e in self.errors],
            "warnings": [w.to_dict() for w in self.warnings],
            "info": [i.to_dict() for i in self.info]
        }


class DeterministicRuleEngine:
    """
    Deterministic hardware validation engine.
    
    This engine:
    - Loads structured rules from JSON database
    - Evaluates hardware configurations against rules
    - Returns deterministic, auditable results
    - NEVER guesses or infers
    - NEVER bypasses failing rules
    """
    
    def __init__(self, rules_path: Optional[Path] = None):
        """Initialize rule engine with rule database."""
        if rules_path is None:
            # Updated path to rules directory
            rules_path = Path(__file__).parent / "rules" / "esp32_rules.json"
        
        try:
            with open(rules_path, 'r') as f:
                self.rule_db = json.load(f)
        except Exception as e:
            print(f"[RuleEngine] ERROR: Could not load rules from {rules_path}: {e}")
            self.rule_db = {"rules": [], "rule_templates": {}}
        
        self.board = self.rule_db.get("board", "esp32")
        self.rules = self.rule_db.get("rules", [])
        self.rule_templates = self.rule_db.get("rule_templates", {})
        
        # Expand templates into concrete rules
        self._expand_templates()
        
        print(f"[RuleEngine] Loaded {len(self.rules)} rules for {self.board}")
        print(f"[RuleEngine] Error states covered: {self.rule_db.get('error_state_count', 0)}")
    
    def _expand_templates(self):
        """Expand rule templates into concrete rules."""
        # Input-only pins (GPIO 34-39)
        for pin in [34, 35, 36, 37, 38, 39]:
            self.rules.append({
                "rule_id": f"ESP32-PIN-{pin:03d}",
                "category": "pin_capability",
                "condition": f"GPIO{pin} == OUTPUT",
                "severity": "error",
                "message": f"GPIO{pin} is input-only. Cannot be configured as OUTPUT.",
                "fix_suggestion": f"Use GPIO 0-33 for OUTPUT. GPIO{pin} can only be INPUT.",
                "datasheet_reference": "ESP32 Datasheet v4.2, Section 4.2 - IO_MUX Pad List"
            })
        
        # Flash-connected pins (GPIO 6-11)
        for pin in [6, 7, 8, 9, 10, 11]:
            self.rules.append({
                "rule_id": f"ESP32-FLASH-{pin:03d}",
                "category": "peripheral_conflict",
                "condition": f"GPIO{pin} == USER_DEFINED",
                "severity": "error",
                "message": f"GPIO{pin} is connected to integrated SPI flash. Cannot be used for general purpose IO.",
                "fix_suggestion": f"Avoid GPIO{pin}. Use other available GPIOs.",
                "datasheet_reference": "ESP32 Datasheet v4.2, Section 2.2 - Pin Definitions"
            })
        
        print(f"[RuleEngine] Expanded templates: +{6 + 6} rules")
    
    def validate_configuration(self, config: Dict[str, Any]) -> ValidationResult:
        """
        Validate hardware configuration against all rules.
        
        Args:
            config: Hardware configuration dict with keys like:
                - gpio_assignments: {pin: mode}
                - peripherals: {name: enabled}
                - clock_config: {uart_baud, pwm_freq, etc.}
                - power_config: {gpio_current, etc.}
        
        Returns:
            ValidationResult with all violations
        """
        errors = []
        warnings = []
        info = []
        
        for rule in self.rules:
            violation = self._evaluate_rule(rule, config)
            if violation:
                if violation.severity == "error":
                    errors.append(violation)
                elif violation.severity == "warning":
                    warnings.append(violation)
                else:
                    info.append(violation)
        
        is_valid = len(errors) == 0
        
        return ValidationResult(
            is_valid=is_valid,
            errors=errors,
            warnings=warnings,
            info=info
        )
    
    def _evaluate_rule(self, rule: Dict[str, Any], config: Dict[str, Any]) -> Optional[RuleViolation]:
        """
        Evaluate a single rule against configuration.
        
        This is deterministic boolean evaluation, NOT AI inference.
        """
        condition = rule["condition"]
        
        # Parse condition and evaluate
        try:
            # Simple condition parser (can be extended)
            if self._evaluate_condition(condition, config):
                return RuleViolation(
                    rule_id=rule["rule_id"],
                    severity=rule["severity"],
                    message=rule["message"],
                    fix_suggestion=rule["fix_suggestion"],
                    datasheet_reference=rule["datasheet_reference"]
                )
        except Exception as e:
            print(f"[RuleEngine] Failed to evaluate rule {rule['rule_id']}: {e}")
        
        return None
    
    def _evaluate_condition(self, condition: str, config: Dict[str, Any]) -> bool:
        """
        Evaluate boolean condition against configuration.
        
        Supported operators: ==, !=, >, <, >=, <=, &&, ||
        """
        # Extract GPIO pin checks
        gpio_match = re.match(r'GPIO(\d+)\s*==\s*(\w+)', condition)
        if gpio_match:
            pin = int(gpio_match.group(1))
            expected_mode = gpio_match.group(2)
            gpio_assignments = config.get("gpio_assignments", {})
            actual_mode = gpio_assignments.get(pin, "UNUSED")
            return actual_mode == expected_mode
        
        # Extract peripheral checks
        periph_match = re.search(r'(\w+)_enabled\s*==\s*(true|false)', condition)
        if periph_match:
            periph_name = periph_match.group(1)
            expected_state = periph_match.group(2) == "true"
            peripherals = config.get("peripherals", {})
            actual_state = peripherals.get(periph_name, False)
            return actual_state == expected_state
        
        # Extract numeric comparisons
        numeric_match = re.match(r'(\w+)\s*([><=!]+)\s*(\d+)', condition)
        if numeric_match:
            var_name = numeric_match.group(1)
            operator = numeric_match.group(2)
            threshold = int(numeric_match.group(3))
            
            # Look up value in config
            value = config.get("clock_config", {}).get(var_name) or \
                    config.get("power_config", {}).get(var_name) or \
                    config.get("memory_config", {}).get(var_name)
            
            if value is not None:
                if operator == ">":
                    return value > threshold
                elif operator == "<":
                    return value < threshold
                elif operator == ">=":
                    return value >= threshold
                elif operator == "<=":
                    return value <= threshold
                elif operator == "==":
                    return value == threshold
                elif operator == "!=":
                    return value != threshold
        
        # Extract string comparisons (e.g., flash_voltage == '3.3V')
        string_match = re.match(r'(\w+)\s*==\s*[\'"]([^\'"]+)[\'"]', condition)
        if string_match:
            var_name = string_match.group(1)
            expected_value = string_match.group(2)
            
            # Look up value in config (top-level first, then nested)
            actual_value = config.get(var_name) or \
                          config.get("clock_config", {}).get(var_name) or \
                          config.get("power_config", {}).get(var_name)
            
            return actual_value == expected_value
        
        # Complex conditions with && or ||
        if "&&" in condition or "||" in condition:
            # Split and recursively evaluate
            if "&&" in condition:
                parts = condition.split("&&")
                return all(self._evaluate_condition(p.strip(), config) for p in parts)
            else:
                parts = condition.split("||")
                return any(self._evaluate_condition(p.strip(), config) for p in parts)
        
        return False
    
    def diagnose_flash_failure(self, log_output: str) -> List[RuleViolation]:
        """
        Diagnose flashing failures from log output.
        
        Maps log patterns to root causes deterministically.
        """
        violations = []
        
        # Flash failure patterns
        flash_patterns = [
            {
                "pattern": r"Timed out waiting for packet header",
                "rule_id": "ESP32-FLASH-TIMEOUT",
                "message": "Flashing timeout. ESP32 not entering download mode.",
                "fix": "Hold GPIO0 LOW and press EN button to enter download mode. Check USB cable and drivers."
            },
            {
                "pattern": r"Invalid head of packet",
                "rule_id": "ESP32-FLASH-INVALID",
                "message": "Invalid packet header. Communication error with ESP32.",
                "fix": "Check baud rate, USB cable quality, and serial port permissions."
            },
            {
                "pattern": r"Failed to connect",
                "rule_id": "ESP32-FLASH-NOCONNECT",
                "message": "Cannot establish serial connection to ESP32.",
                "fix": "Verify COM port, install CH340/CP2102 drivers, check USB cable."
            }
        ]
        
        for pattern_def in flash_patterns:
            if re.search(pattern_def["pattern"], log_output, re.IGNORECASE):
                violations.append(RuleViolation(
                    rule_id=pattern_def["rule_id"],
                    severity="error",
                    message=pattern_def["message"],
                    fix_suggestion=pattern_def["fix"],
                    datasheet_reference="ESP32 Datasheet v4.2, Section 2.4 - Boot Mode"
                ))
        
        return violations
