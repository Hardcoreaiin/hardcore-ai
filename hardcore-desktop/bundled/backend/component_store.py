"""
Component Store - Store and retrieve component profiles per project.

Stores profiles as JSON files in the workspace directory.
"""

import os
import json
from typing import Dict, Any, Optional, List
from pathlib import Path
from datetime import datetime

from datasheet_parser import ComponentProfile


class ComponentStore:
    """
    Store and retrieve component profiles per project.
    
    Storage structure:
    workspace/{project_id}/components/{component_name}.json
    """
    
    def __init__(self, workspace_root: Optional[Path] = None):
        if workspace_root is None:
            workspace_root = Path(__file__).parent / "workspace"
        self.workspace_root = Path(workspace_root)
        self.workspace_root.mkdir(parents=True, exist_ok=True)
        print(f"[ComponentStore] Initialized with workspace: {self.workspace_root}")
    
    def _get_components_dir(self, project_id: str) -> Path:
        """Get the components directory for a project."""
        components_dir = self.workspace_root / project_id / "components"
        components_dir.mkdir(parents=True, exist_ok=True)
        return components_dir
    
    def _sanitize_name(self, name: str) -> str:
        """Sanitize component name for use as filename."""
        # Remove invalid characters
        safe_name = "".join(c if c.isalnum() or c in "-_" else "_" for c in name)
        return safe_name.lower()
    
    def save_profile(self, project_id: str, profile: ComponentProfile) -> str:
        """
        Save a component profile to storage.
        
        Args:
            project_id: Project identifier
            profile: ComponentProfile to save
        
        Returns:
            Path to saved file
        """
        components_dir = self._get_components_dir(project_id)
        safe_name = self._sanitize_name(profile.name)
        file_path = components_dir / f"{safe_name}.json"
        
        # Add save timestamp
        profile_dict = profile.to_dict()
        profile_dict['_saved_at'] = datetime.now().isoformat()
        
        with open(file_path, 'w', encoding='utf-8') as f:
            json.dump(profile_dict, f, indent=2)
        
        print(f"[ComponentStore] Saved profile: {profile.name} -> {file_path}")
        return str(file_path)
    
    def get_profile(self, project_id: str, component_name: str) -> Optional[ComponentProfile]:
        """
        Get a specific component profile.
        
        Args:
            project_id: Project identifier
            component_name: Component name to retrieve
        
        Returns:
            ComponentProfile or None if not found
        """
        components_dir = self._get_components_dir(project_id)
        safe_name = self._sanitize_name(component_name)
        file_path = components_dir / f"{safe_name}.json"
        
        if not file_path.exists():
            return None
        
        with open(file_path, 'r', encoding='utf-8') as f:
            data = json.load(f)
        
        # Remove metadata fields before parsing
        data.pop('_saved_at', None)
        
        return ComponentProfile.from_dict(data)
    
    def list_profiles(self, project_id: str) -> List[Dict[str, Any]]:
        """
        List all component profiles for a project.
        
        Args:
            project_id: Project identifier
        
        Returns:
            List of profile summaries (name, type, source_file)
        """
        components_dir = self._get_components_dir(project_id)
        
        profiles = []
        for file_path in components_dir.glob("*.json"):
            try:
                with open(file_path, 'r', encoding='utf-8') as f:
                    data = json.load(f)
                profiles.append({
                    "name": data.get("name", file_path.stem),
                    "type": data.get("type", "unknown"),
                    "source_file": data.get("source_file"),
                    "interfaces": data.get("interfaces", []),
                    "pin_count": len(data.get("pins", [])),
                    "file": file_path.name
                })
            except Exception as e:
                print(f"[ComponentStore] Error reading {file_path}: {e}")
        
        return profiles
    
    def get_all_profiles(self, project_id: str) -> List[ComponentProfile]:
        """
        Get all component profiles for a project as full objects.
        
        Args:
            project_id: Project identifier
        
        Returns:
            List of ComponentProfile objects
        """
        components_dir = self._get_components_dir(project_id)
        
        profiles = []
        for file_path in components_dir.glob("*.json"):
            try:
                with open(file_path, 'r', encoding='utf-8') as f:
                    data = json.load(f)
                data.pop('_saved_at', None)
                profiles.append(ComponentProfile.from_dict(data))
            except Exception as e:
                print(f"[ComponentStore] Error reading {file_path}: {e}")
        
        return profiles
    
    def delete_profile(self, project_id: str, component_name: str) -> bool:
        """
        Delete a component profile.
        
        Args:
            project_id: Project identifier
            component_name: Component name to delete
        
        Returns:
            True if deleted, False if not found
        """
        components_dir = self._get_components_dir(project_id)
        safe_name = self._sanitize_name(component_name)
        file_path = components_dir / f"{safe_name}.json"
        
        if file_path.exists():
            file_path.unlink()
            print(f"[ComponentStore] Deleted profile: {component_name}")
            return True
        
        return False
    
    def format_for_prompt(self, project_id: str) -> str:
        """
        Format all component profiles for injection into AI prompt.
        
        Args:
            project_id: Project identifier
        
        Returns:
            Formatted string for AI prompt
        """
        profiles = self.get_all_profiles(project_id)
        
        if not profiles:
            return ""
        
        lines = [
            "=== COMPONENT DATASHEETS (GROUND TRUTH) ===",
            "The following component specifications are from uploaded datasheets.",
            "You MUST use these EXACT specifications. Do NOT invent or assume values.",
            ""
        ]
        
        for profile in profiles:
            lines.append(f"### {profile.name.upper()} ({profile.type})")
            
            if profile.voltage_range:
                vr = profile.voltage_range
                lines.append(f"Operating Voltage: {vr.min}V - {vr.max}V")
            
            if profile.logic_level:
                ll = profile.logic_level
                lines.append(f"Logic Level: {ll.min}V - {ll.max}V")
            
            if profile.interfaces:
                lines.append(f"Interfaces: {', '.join(profile.interfaces)}")
            
            if profile.i2c_address:
                lines.append(f"I2C Address: {profile.i2c_address}")
            
            if profile.pins:
                lines.append("Pins:")
                for pin in profile.pins:
                    pin_str = f"  - {pin.name}"
                    if pin.number:
                        pin_str += f" (Pin {pin.number})"
                    pin_str += f": {pin.function} [{pin.type}]"
                    lines.append(pin_str)
            
            if profile.timing:
                t = profile.timing
                if t.startup_delay_ms:
                    lines.append(f"Startup Delay: {t.startup_delay_ms}ms")
                if t.min_pulse_width_us:
                    lines.append(f"Min Pulse Width: {t.min_pulse_width_us}µs")
            
            if profile.external_components:
                lines.append(f"Required: {', '.join(profile.external_components)}")
            
            lines.append("")
        
        lines.append("=== END COMPONENT DATASHEETS ===")
        lines.append("Use ONLY these specifications. Do NOT hallucinate pins or addresses.")
        
        return "\n".join(lines)


# Quick test
if __name__ == "__main__":
    store = ComponentStore()
    print("ComponentStore ready")
    print("Profiles in 'test_project':", store.list_profiles("test_project"))
