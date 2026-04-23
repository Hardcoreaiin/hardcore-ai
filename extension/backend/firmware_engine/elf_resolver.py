import os
from elftools.elf.elffile import ELFFile
from elftools.elf.sections import SymbolTableSection

class ELFResolver:
    """ 
    Resolves addresses to function names and line numbers using ELF files. 
    Expert implementation using both Symbol Table and DWARF debug info.
    """
    
    def __init__(self, elf_path):
        self.elf_path = elf_path
        if not os.path.exists(elf_path):
            raise FileNotFoundError(f"ELF file not found: {elf_path}")
        
    def resolve_address(self, address):
        """ 
        Translates a hex address to a function name and source location. 
        """
        result = {
            "name": "Unknown",
            "address": hex(address),
            "file": "N/A",
            "line": "N/A"
        }

        with open(self.elf_path, 'rb') as f:
            elffile = ELFFile(f)
            
            # 1. Try DWARF first for pinpoint accuracy (file + line + function)
            if elffile.has_dwarf_info():
                dwarf_info = elffile.get_dwarf_info()
                src_loc = self._get_source_location_dwarf(dwarf_info, address)
                if src_loc:
                    result.update(src_loc)
                    # If we found it via DWARF, we're likely done
                    if result["name"] != "Unknown":
                        return result

            # 2. Fallback to Symbol Table for function name only
            for section in elffile.iter_sections():
                if not isinstance(section, SymbolTableSection):
                    continue
                
                for symbol in section.iter_symbols():
                    if symbol['st_value'] <= address < symbol['st_value'] + symbol['st_size']:
                        result["name"] = symbol.name
                        result["address"] = hex(symbol['st_value'])
                        break
        
        return result

    def _get_source_location_dwarf(self, dwarf_info, address):
        """ Internal helper to traverse DWARF info. """
        for cu in dwarf_info.iter_CUs():
            line_program = dwarf_info.line_program_for_CU(cu)
            prev_state = None
            for entry in line_program.get_entries():
                state = entry.state
                if state is None: continue
                if state.end_sequence: 
                    prev_state = None
                    continue
                
                if prev_state and prev_state.address <= address < state.address:
                    file_name = line_program['file_entry'][prev_state.file - 1].name.decode('utf-8')
                    # Find function name in this CU
                    func_name = "Unknown"
                    for die in cu.iter_DIEs():
                        if die.tag == 'DW_TAG_subprogram':
                            try:
                                low_pc = die.attributes['DW_AT_low_pc'].value
                                # High PC can be an offset or an address
                                high_pc_attr = die.attributes['DW_AT_high_pc']
                                if high_pc_attr.form == 'DW_FORM_addr':
                                    high_pc = high_pc_attr.value
                                else:
                                    high_pc = low_pc + high_pc_attr.value
                                
                                if low_pc <= address < high_pc:
                                    func_name = die.attributes['DW_AT_name'].value.decode('utf-8')
                                    break
                            except KeyError:
                                continue
                                
                    return {
                        "file": os.path.basename(file_name),
                        "line": prev_state.line,
                        "name": func_name
                    }
                prev_state = state
        return None

if __name__ == "__main__":
    import sys
    if len(sys.argv) < 3:
        print("Usage: python elf_resolver.py <elf_file> <address>")
        sys.exit(1)
        
    resolver = ELFResolver(sys.argv[1])
    addr = int(sys.argv[2], 16)
    result = resolver.resolve_address(addr)
    print(f"Resolved: {result['name']}() at {result['file']}:{result['line']}")
