import logging
from typing import Dict, Any

logger = logging.getLogger(__name__)

class DiagramGenerator:
    """
    Generates SVG/Mermaid block diagrams for hardware architectures.
    """
    
    @staticmethod
    def generate_system_block_diagram(hardware_blocks: list, interfaces: list) -> str:
        """
        Generates a Mermaid.js block diagram representing the system architecture.
        """
        
        # Start the Mermaid graph
        mermaid_lines = ["graph TD"]
        
        # Central node (usually the MPU/MCU)
        core_node = None
        for block in hardware_blocks:
            if "i.MX" in block.get("name", "") or "STM" in block.get("name", "") or "ESP" in block.get("name", "") or "Processor" in block.get("name", "") or "MCU" in block.get("name", ""):
                core_node = block.get("name")
                break
                
        if not core_node and hardware_blocks:
            core_node = hardware_blocks[0].get("name")
            
        if not core_node:
            core_node = "Main Processor"
            
        # Format the core node ID (remove spaces/special chars)
        core_id = core_node.replace(" ", "_").replace(".", "_").replace("-", "_")
        mermaid_lines.append(f'    {core_id}["{core_node}"]')
        
        # Add peripheral nodes and connections based on interfaces
        added_nodes = set([core_id])
        
        for interface in interfaces:
            comp_name = interface.get("component", "Unknown Component")
            bus = interface.get("bus", "GPIO")
            
            comp_id = comp_name.replace(" ", "_").replace(".", "_").replace("-", "_")
            
            if comp_id not in added_nodes:
                mermaid_lines.append(f'    {comp_id}["{comp_name}"]')
                added_nodes.add(comp_id)
                
            # Connect the core processor to the component via the bus
            mermaid_lines.append(f'    {core_id} -- "{bus}" --> {comp_id}')
            
        # Add any remaining hardware blocks that weren't in the interfaces list
        for block in hardware_blocks:
            block_name = block.get("name", "")
            if block_name != core_node:
                block_id = block_name.replace(" ", "_").replace(".", "_").replace("-", "_")
                if block_id not in added_nodes:
                    mermaid_lines.append(f'    {block_id}["{block_name}"]')
                    added_nodes.add(block_id)
                    # Generic connection if we don't know the bus
                    mermaid_lines.append(f'    {core_id} --- {block_id}')

        # Add styling to make it look professional
        mermaid_lines.append(f'    classDef default fill:#111,stroke:#333,stroke-width:2px,color:#ddd;')
        mermaid_lines.append(f'    classDef core fill:#1e3a8a,stroke:#60a5fa,stroke-width:3px,color:#fff;') # blue-900 fill, blue-400 stroke
        mermaid_lines.append(f'    class {core_id} core;')

        return "\n".join(mermaid_lines)
    
    @staticmethod
    def generate_boot_chain_diagram(boot_stages: list) -> str:
        """
        Generates a Mermaid.js flow diagram representing the boot sequence.
        """
        mermaid_lines = ["graph TD"]
        
        prev_id = None
        for i, stage in enumerate(boot_stages):
            stage_name = stage.get("name", f"Stage {i+1}")
            stage_id = f"stage_{i}"
            
            # Format the node with extra info if available (like security verification)
            desc = stage.get("description", "")
            if desc:
                # Truncate description for the box
                desc = desc[:30] + "..." if len(desc) > 30 else desc
                node_label = f"{stage_name}<br/><small><i>{desc}</i></small>"
            else:
                node_label = stage_name
                
            mermaid_lines.append(f'    {stage_id}["{node_label}"]')
            
            if prev_id:
                # Add transition (can add verification status if needed)
                mermaid_lines.append(f'    {prev_id} --> {stage_id}')
                
            prev_id = stage_id
            
        mermaid_lines.append('    classDef default fill:#111,stroke:#333,stroke-width:2px,color:#ddd;')
        return "\n".join(mermaid_lines)
