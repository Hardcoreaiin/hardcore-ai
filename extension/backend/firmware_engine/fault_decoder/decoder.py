import json
import os

class HardFaultDecoder:
    """ Decodes Cortex-M HardFault registers into human-readable format. """
    
    def __init__(self, rules_path):
        with open(rules_path, 'r') as f:
            self.rules = json.load(f)
            
    def decode(self, cfsr, hfsr, mmfar, bfar):
        results = {
            "fault_type": "None",
            "details": [],
            "address_info": {}
        }
        
        # Decode HFSR
        hfsr_rules = self.rules.get("HFSR", {})
        for name, info in hfsr_rules.items():
            mask = info[0]
            desc = info[1]
            if hfsr & mask:
                results["details"].append({"name": name, "description": desc, "register": "HFSR"})
                if name == "FORCED":
                    results["fault_type"] = "Escalated to HardFault"
                elif name == "VECTBL":
                    results["fault_type"] = "Vector Table Read Fault"

        # Decode CFSR sub-registers (MMFSR, BFSR, UFSR)
        cfsr_rules = self.rules.get("CFSR", {})
        for subgroup, sub_rules in cfsr_rules.items():
            for name, info in sub_rules.items():
                mask = info[0]
                desc = info[1]
                if cfsr & mask:
                    results["details"].append({"name": name, "description": desc, "register": subgroup})
                    
        # Check for address validity
        if cfsr & 0x80: # MMARVALID
            results["address_info"]["MemManage Address"] = hex(mmfar)
        if cfsr & 0x8000: # BFARVALID
            results["address_info"]["BusFault Address"] = hex(bfar)
            
        return results

if __name__ == "__main__":
    # Test with a sample CFSR=0x010000 (UNDEFINSTR)
    rules_path = os.path.join(os.path.dirname(__file__), "..", "..", "rule_engine", "fault_rules.json")
    decoder = HardFaultDecoder(rules_path)
    res = decoder.decode(0x010000, 0x40000000, 0, 0)
    print(json.dumps(res, indent=4))
