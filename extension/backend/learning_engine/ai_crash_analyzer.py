import os
import json
import re
import asyncio
from typing import Dict, Any, Optional, List
import logging

# Set up logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger("AICrashAnalyzer")

# Try to import GeminiAPI from the orchestrator directory
import sys
current_file_dir = os.path.dirname(os.path.abspath(__file__))
orchestrator_dir = os.path.abspath(os.path.join(current_file_dir, "../Hardcore.ai/orchestrator"))
if orchestrator_dir not in sys.path:
    sys.path.append(orchestrator_dir)

try:
    from gemini_api import GeminiAPI
except ImportError:
    # Fallback/Mock for local testing if needed
    class GeminiAPI:
        def __init__(self, **kwargs): pass
        async def generate_content_async(self, prompt, **kwargs): return ""

class AICrashAnalyzer:
    """
    Secondary reasoning system for HARDCOREAI.
    Uses LLM to provide semantic explanation and fixes for hardware faults.
    """

    PROMPT_TEMPLATE = """
You are HARDCOREAI, a production-grade embedded crash intelligence system.
Your task is to provide an AI Intelligence Layer for a detected hardware crash.
You will receive a deterministic diagnostic report as ground truth. 
Your job is to EXPLAIN the root cause, RECOGNIZE bug patterns, and SUGGEST specific fix steps.

---
# 🚨 GROUND TRUTH (Deterministic Data)
ARCHITECTURE: {architecture}
FAULT TYPE: {fault_type}
CONFIDENCE: {confidence}
MEMORY REGION: {memory_region_desc}
LIKELY SCENARIO: {likely_scenario}

# 🧠 REGISTER CONTEXT
{registers_json}

# 📂 SYMBOLIC CONTEXT
FUNCTION: {function_name}
LOCATION: {file_path}:{line_number}
STACK TRACE: {stack_trace}

# 💻 CODE SNIPPET (±5 lines)
```c
{code_snippet}
```
---

# 🎯 REQUIREMENTS
1. Use the deterministic findings as ground truth (do not contradict them).
2. STACK ALIGNMENT & PERIPHERALS: Explain context specific to {architecture}.
3. NO HALLUCINATION: Use the exact register values (PC, LR, BFAR) provided. If a memory address is in BFAR, that is the EXACT point of failure.
4. If a peripheral address is detected (e.g., 0x4000... or 0x5000...), verify if its clock is enabled in the RCC.
5. If the fault is in SRAM (0x2000...), check for buffer overflows or stack fragmentation.

# 🚨 OUTPUT FORMAT (STRICT JSON ONLY)
Return exactly this JSON schema:
{{
  "root_cause_explanation": "A high-level explanation of what specifically failed and why, referencing the exact address in BFAR.",
  "confidence": "high|medium|low",
  "fix_steps": [
    "Step 1 to fix the bug",
    "Step 2 to fix the bug"
  ],
  "bug_pattern": "Name of the pattern (e.g., Clock Gating Issue, Stack Smash)",
  "risk_level": "critical|high|medium"
}}
"""

    def __init__(self, api_key: Optional[str] = None):
        try:
            self.gemini = GeminiAPI(api_key=api_key)
            self.api_available = True
        except Exception as e:
            logger.warning(f"AI Analyzer disabled: {e}")
            self.api_available = False

    async def analyze(self, diagnostic_report: Dict[str, Any], code_snippet: str = "N/A") -> Optional[Dict[str, Any]]:
        """
        Asynchronously call AI to analyze the crash.
        """
        if not self.api_available:
            return None

        prompt = self._build_prompt(diagnostic_report, code_snippet)
        
        try:
            # Set a timeout for AI response
            raw_response = await self.gemini.generate_content_async(
                prompt, 
                temperature=0.1 # Low temperature for factual consistency
            )
            
            if not raw_response:
                return None

            return self._parse_response(raw_response)
        except Exception as e:
            logger.error(f"AI Analysis failed: {e}")
            return None

    def _build_prompt(self, report: Dict[str, Any], code_snippet: str) -> str:
        # Extract fields for prompt
        where = report.get("where", {})
        mem = report.get("memory_analysis", {})
        
        return self.PROMPT_TEMPLATE.format(
            architecture=report.get("arch", "Unknown"),
            fault_type=report.get("fault_type", "Unknown"),
            confidence=report.get("confidence", "Unknown"),
            memory_region_desc=f"{mem.get('region', 'N/A')} ({mem.get('description', 'N/A')})",
            likely_scenario=report.get("likely_scenario", "N/A"),
            registers_json=json.dumps(report.get("raw_registers", {}), indent=2),
            function_name=where.get("function", "Unknown"),
            file_path=where.get("file", "N/A"),
            line_number=where.get("line", "N/A"),
            stack_trace=" -> ".join(report.get("stack_trace_names", ["Unknown"])),
            code_snippet=code_snippet
        )

    def _parse_response(self, text: str) -> Optional[Dict[str, Any]]:
        """Extract and parse JSON from LLM response."""
        try:
            # Look for JSON block
            json_match = re.search(r'```json\s*(.*?)\s*```', text, re.DOTALL)
            if json_match:
                content = json_match.group(1).strip()
            else:
                # Try to find something that looks like JSON if no code block
                json_match = re.search(r'\{.*\}', text, re.DOTALL)
                content = json_match.group(0).strip() if json_match else text.strip()

            data = json.loads(content)
            
            # Basic schema validation
            required = ["root_cause_explanation", "confidence", "fix_steps", "bug_pattern", "risk_level"]
            if all(k in data for k in required):
                return data
            return None
        except Exception as e:
            logger.error(f"Failed to parse AI response: {e}\nRaw: {text[:200]}")
            return None

if __name__ == "__main__":
    # Quick manual test mock
    async def test():
        analyzer = AICrashAnalyzer()
        mock_report = {
            "arch": "Cortex-M4",
            "fault_type": "Bus Access Error",
            "confidence": "Deterministic",
            "memory_analysis": {"region": "Peripheral", "description": "GPIOA Range"},
            "likely_scenario": "Attempted to access peripheral with disabled clock.",
            "where": {"function": "main", "file": "main.c", "line": 42},
            "raw_registers": {"pc": "0x40020000", "lr": "0x08001234"},
            "stack_trace_names": ["Reset_Handler", "main"]
        }
        res = await analyzer.analyze(mock_report, "GPIOA->ODR = 1;")
        print(json.dumps(res, indent=2))

    # asyncio.run(test()) # Uncomment to test locally
