"""
Visualization Engine - Layout Generator
Generates JSON data for frontend component rendering.
"""

from typing import Dict, List, Any

class LayoutGenerator:
    """Generates visual layout for ESP32 and components."""
    
    ESP32_PIN_POSITIONS = {
        # Left side (USB up)
        "3V3": {"x": 10, "y": 10},
        "GND": {"x": 10, "y": 20},
        "D15": {"x": 10, "y": 30},
        "D2":  {"x": 10, "y": 40},
        "D4":  {"x": 10, "y": 50},
        "RX2": {"x": 10, "y": 60},
        "TX2": {"x": 10, "y": 70},
        "D5":  {"x": 10, "y": 80},
        "D18": {"x": 10, "y": 90},
        "D19": {"x": 10, "y": 100},
        "D21": {"x": 10, "y": 110},
        "RX0": {"x": 10, "y": 120},
        "TX0": {"x": 10, "y": 130},
        "D22": {"x": 10, "y": 140},
        "D23": {"x": 10, "y": 150},
        
        # Right side
        "VIN": {"x": 90, "y": 10},
        "GND_R": {"x": 90, "y": 20},
        "D13": {"x": 90, "y": 30},
        "D12": {"x": 90, "y": 40},
        "D14": {"x": 90, "y": 50},
        "D27": {"x": 90, "y": 60},
        "D26": {"x": 90, "y": 70},
        "D25": {"x": 90, "y": 80},
        "D33": {"x": 90, "y": 90},
        "D32": {"x": 90, "y": 100},
        "D35": {"x": 90, "y": 110},
        "D34": {"x": 90, "y": 120},
        "VN":  {"x": 90, "y": 130},
        "VP":  {"x": 90, "y": 140},
        "EN":  {"x": 90, "y": 150}
    }
    
    def generate_layout(self, components: List[Dict[str, Any]]) -> Dict[str, Any]:
        """
        Generate layout for a list of components.
        
        Args:
            components: List of components with connections.
            Example: [{"name": "L298N", "pins": {"IN1": 15, "IN2": 2}}]
            
        Returns:
            JSON layout for frontend.
        """
        layout = {
            "board": "ESP32",
            "components": [],
            "wires": []
        }
        
        # Simple auto-layout algorithm
        x_offset = 200
        y_offset = 50
        
        for i, comp in enumerate(components):
            comp_name = comp.get("name", f"Component_{i}")
            comp_type = comp.get("type", "Generic")
            
            # Position component
            comp_entry = {
                "id": f"comp_{i}",
                "name": comp_name,
                "type": comp_type,
                "x": x_offset,
                "y": y_offset + (i * 150)
            }
            layout["components"].append(comp_entry)
            
            # Generate wires
            pins = comp.get("pins", {})
            for pin_name, mcu_pin in pins.items():
                wire = {
                    "from_component": comp_entry["id"],
                    "from_pin": pin_name,
                    "to_board": "ESP32",
                    "to_pin": f"D{mcu_pin}",  # Assumes Dx format
                    "color": self._get_wire_color(pin_name)
                }
                layout["wires"].append(wire)
                
        return layout
    
    def _get_wire_color(self, pin_name: str) -> str:
        name = pin_name.upper()
        if "VCC" in name or "5V" in name or "3V3" in name:
            return "#FF0000"  # Red
        if "GND" in name:
            return "#000000"  # Black
        if "RX" in name:
            return "#FFFF00"  # Yellow
        if "TX" in name:
            return "#00FF00"  # Green
        return "#0000FF"  # Blue (Signal)
