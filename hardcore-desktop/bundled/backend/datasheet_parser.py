"""
Datasheet Parser - PDF parsing and component profile extraction.

Uses pdfplumber for text extraction and Gemini for structured data extraction.
"""

import os
import json
import re
from typing import Dict, Any, Optional, List
from dataclasses import dataclass, field, asdict
from pathlib import Path

# Try importing PDF libraries
try:
    import pdfplumber
    PDF_AVAILABLE = True
except ImportError:
    PDF_AVAILABLE = False
    print("[DatasheetParser] pdfplumber not installed. PDF parsing disabled.")

import requests


@dataclass
class PinInfo:
    """Single pin from datasheet."""
    name: str
    number: Optional[int] = None
    function: str = ""
    type: str = "GPIO"  # GPIO, PWM, I2C, SPI, UART, ADC, POWER, GND


@dataclass
class TimingInfo:
    """Timing constraints from datasheet."""
    startup_delay_ms: Optional[float] = None
    min_pulse_width_us: Optional[float] = None
    max_frequency_hz: Optional[float] = None
    sampling_time_us: Optional[float] = None


@dataclass
class VoltageRange:
    """Voltage specification."""
    min: float = 0
    max: float = 0
    typical: Optional[float] = None
    unit: str = "V"


@dataclass
class ComponentProfile:
    """Complete component profile extracted from datasheet."""
    name: str
    type: str  # sensor, motor_driver, display, ic, module, etc.
    manufacturer: Optional[str] = None
    part_number: Optional[str] = None
    
    # Electrical specs
    voltage_range: Optional[VoltageRange] = None
    logic_level: Optional[VoltageRange] = None
    max_current_ma: Optional[float] = None
    
    # Pins
    pins: List[PinInfo] = field(default_factory=list)
    
    # Communication
    interfaces: List[str] = field(default_factory=list)  # GPIO, PWM, I2C, SPI, UART, ADC
    i2c_address: Optional[str] = None
    spi_mode: Optional[int] = None
    
    # Timing
    timing: Optional[TimingInfo] = None
    
    # Additional info
    external_components: List[str] = field(default_factory=list)
    typical_applications: List[str] = field(default_factory=list)
    notes: List[str] = field(default_factory=list)
    
    # Metadata
    source_file: Optional[str] = None
    extraction_confidence: float = 0.0
    
    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary for JSON storage."""
        result = asdict(self)
        # Clean up None values
        return {k: v for k, v in result.items() if v is not None}
    
    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> 'ComponentProfile':
        """Create from dictionary."""
        # Handle nested dataclasses
        if 'voltage_range' in data and data['voltage_range']:
            data['voltage_range'] = VoltageRange(**data['voltage_range'])
        if 'logic_level' in data and data['logic_level']:
            data['logic_level'] = VoltageRange(**data['logic_level'])
        if 'timing' in data and data['timing']:
            data['timing'] = TimingInfo(**data['timing'])
        if 'pins' in data:
            data['pins'] = [PinInfo(**p) if isinstance(p, dict) else p for p in data['pins']]
        return cls(**data)


class DatasheetParser:
    """
    Parse component datasheets and extract structured profiles.
    
    Uses pdfplumber for text extraction and Gemini for intelligent structuring.
    """
    
    EXTRACTION_PROMPT = """You are analyzing a component datasheet. Extract the following information in JSON format.

DATASHEET TEXT:
{datasheet_text}

Extract this JSON (use null for unknown values, do NOT invent data):

{{
  "name": "component name",
  "type": "sensor|motor_driver|display|ic|module|relay|actuator|communication|other",
  "manufacturer": "manufacturer name or null",
  "part_number": "part number or null",
  "voltage_range": {{"min": number, "max": number, "typical": number or null, "unit": "V"}},
  "logic_level": {{"min": number, "max": number, "unit": "V"}} or null,
  "max_current_ma": number or null,
  "pins": [
    {{"name": "pin name", "number": pin_number or null, "function": "description", "type": "GPIO|PWM|I2C|SPI|UART|ADC|POWER|GND"}}
  ],
  "interfaces": ["GPIO", "PWM", "I2C", "SPI", "UART", "ADC"],
  "i2c_address": "0xXX" or null,
  "timing": {{
    "startup_delay_ms": number or null,
    "min_pulse_width_us": number or null
  }} or null,
  "external_components": ["pullup resistors", "decoupling caps", etc.] or [],
  "typical_applications": ["application 1", "application 2"] or [],
  "notes": ["important note 1", "warning 2"] or []
}}

CRITICAL RULES:
- Extract ONLY what is explicitly stated in the datasheet
- Use null for any value not clearly stated
- Do NOT invent or assume pin numbers
- Do NOT hallucinate register addresses
- Include all pins mentioned in the datasheet
- For motor drivers: identify enable, direction, and motor output pins
- For sensors: identify data, clock, and interrupt pins

Return ONLY the JSON, no explanation."""

    def __init__(self, api_key: Optional[str] = None):
        self.api_key = api_key or os.getenv("LLM_API_KEY", "")
        self.model = "gemini-2.0-flash"
    
    def parse_pdf(self, file_path: str, max_pages: int = 10) -> str:
        """
        Extract text from PDF file.
        
        Args:
            file_path: Path to PDF file
            max_pages: Maximum pages to parse (first N pages usually have key info)
        
        Returns:
            Extracted text
        """
        if not PDF_AVAILABLE:
            raise RuntimeError("pdfplumber not installed. Run: pip install pdfplumber")
        
        path = Path(file_path)
        if not path.exists():
            raise FileNotFoundError(f"PDF not found: {file_path}")
        
        if not path.suffix.lower() == '.pdf':
            raise ValueError(f"Not a PDF file: {file_path}")
        
        print(f"[DatasheetParser] Parsing PDF: {path.name}")
        
        text_parts = []
        with pdfplumber.open(file_path) as pdf:
            pages_to_parse = min(len(pdf.pages), max_pages)
            print(f"[DatasheetParser] Parsing {pages_to_parse} of {len(pdf.pages)} pages")
            
            for i, page in enumerate(pdf.pages[:pages_to_parse]):
                page_text = page.extract_text() or ""
                if page_text.strip():
                    text_parts.append(f"--- PAGE {i+1} ---\n{page_text}")
        
        full_text = "\n\n".join(text_parts)
        print(f"[DatasheetParser] Extracted {len(full_text)} characters")
        
        return full_text
    
    def extract_component_profile(self, datasheet_text: str, source_file: Optional[str] = None) -> ComponentProfile:
        """
        Use Gemini to extract structured component profile from datasheet text.
        
        Args:
            datasheet_text: Raw text from datasheet
            source_file: Original filename (for reference)
        
        Returns:
            ComponentProfile with extracted data
        """
        if not self.api_key:
            raise RuntimeError("No API key configured for profile extraction")
        
        # Truncate text if too long (Gemini has context limits)
        max_chars = 30000
        if len(datasheet_text) > max_chars:
            print(f"[DatasheetParser] Truncating text from {len(datasheet_text)} to {max_chars} chars")
            datasheet_text = datasheet_text[:max_chars]
        
        prompt = self.EXTRACTION_PROMPT.format(datasheet_text=datasheet_text)
        
        print(f"[DatasheetParser] Calling Gemini for profile extraction...")
        
        # Call Gemini API
        endpoint = f"https://generativelanguage.googleapis.com/v1beta/models/{self.model}:generateContent?key={self.api_key}"
        
        payload = {
            "contents": [{"parts": [{"text": prompt}]}],
            "generationConfig": {
                "temperature": 0.1,  # Low temperature for accuracy
                "maxOutputTokens": 4096
            }
        }
        
        try:
            response = requests.post(endpoint, json=payload, timeout=60)
            response.raise_for_status()
            result = response.json()
            
            # Extract text from response
            text = result.get("candidates", [{}])[0].get("content", {}).get("parts", [{}])[0].get("text", "")
            
            # Parse JSON from response
            json_match = re.search(r'\{[\s\S]*\}', text)
            if not json_match:
                raise ValueError("No JSON found in Gemini response")
            
            profile_data = json.loads(json_match.group())
            
            # Add metadata
            profile_data['source_file'] = source_file
            profile_data['extraction_confidence'] = 0.8  # Default confidence
            
            profile = ComponentProfile.from_dict(profile_data)
            
            print(f"[DatasheetParser] ✓ Extracted profile for: {profile.name}")
            print(f"[DatasheetParser]   Type: {profile.type}")
            print(f"[DatasheetParser]   Pins: {len(profile.pins)}")
            print(f"[DatasheetParser]   Interfaces: {profile.interfaces}")
            
            return profile
            
        except requests.RequestException as e:
            print(f"[DatasheetParser] API error: {e}")
            raise RuntimeError(f"Failed to call Gemini API: {e}")
        except json.JSONDecodeError as e:
            print(f"[DatasheetParser] JSON parse error: {e}")
            raise ValueError(f"Failed to parse Gemini response as JSON: {e}")
    
    def parse_and_extract(self, pdf_path: str) -> ComponentProfile:
        """
        Complete pipeline: parse PDF and extract component profile.
        
        Args:
            pdf_path: Path to PDF datasheet
        
        Returns:
            ComponentProfile with extracted data
        """
        text = self.parse_pdf(pdf_path)
        profile = self.extract_component_profile(text, source_file=Path(pdf_path).name)
        return profile


# Quick test
if __name__ == "__main__":
    parser = DatasheetParser()
    print("DatasheetParser ready. PDF support:", PDF_AVAILABLE)
