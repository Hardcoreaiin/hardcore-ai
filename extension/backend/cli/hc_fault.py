import argparse
import sys
import os
import json
import time

# Add parent directory to path for imports
sys.path.append(os.path.join(os.path.dirname(__file__), ".."))

from firmware_engine.elf_resolver import ELFResolver # type: ignore
from firmware_engine.fault_decoder.decoder import HardFaultDecoder # type: ignore
from cli.diagnostic_engine import DiagnosticEngine

def main():
    parser = argparse.ArgumentParser(description="HARDCOREAI Platform CLI")
    subparsers = parser.add_subparsers(dest="command", help="Available commands")

    # Debug subcommand
    debug_parser = subparsers.add_parser("debug", help="Debugging tools")
    debug_subparsers = debug_parser.add_subparsers(dest="subcommand", help="Debug subcommands")

    # Hardfault subcommand
    hf_parser = debug_subparsers.add_parser("hardfault", help="Decode ARM Cortex-M HardFault")
    hf_parser.add_argument("--cfsr", help="Configurable Fault Status Register")
    hf_parser.add_argument("--hfsr", help="HardFault Status Register")
    hf_parser.add_argument("--mmfar", help="MemManage Fault Address Register")
    hf_parser.add_argument("--bfar", help="BusFault Address Register")
    hf_parser.add_argument("--pc", help="Program Counter at crash")
    hf_parser.add_argument("--elf", help="Path to ELF file for symbol resolution")
    hf_parser.add_argument("--input", help="Path to JSON input file")
    hf_parser.add_argument("--json", action="store_true", help="Output only JSON")
    
    # Live subcommand
    live_parser = debug_subparsers.add_parser("live", help="Live monitor hardware for faults")
    live_parser.add_argument("--interface", default="interface/stlink.cfg", help="OpenOCD interface config")
    live_parser.add_argument("--target", default="target/stm32f4x.cfg", help="OpenOCD target config")
    live_parser.add_argument("--elf", help="Path to ELF for symbol resolution")
    live_parser.add_argument("--uart", help="UART port (e.g. COM3 or /dev/ttyUSB0)")

    args = parser.parse_args()

    if args.command == "debug":
        if args.subcommand == "hardfault":
            process_hardfault(args)
        elif args.subcommand == "live":
            process_live(args)
    else:
        parser.print_help()
        sys.exit(1)

def process_live(args):
    from hardware.orchestrator import HardwareOrchestrator # type: ignore
    
    config = {
        "interface": args.interface,
        "target": args.target,
        "elf_path": args.elf,
        "uart_port": args.uart
    }
    
    orchestrator = HardwareOrchestrator(config)
    
    def on_fault(report):
        # In live mode, we output JSON for the extension to consume
        print(f"FAULT_REPORT:{json.dumps(report)}")

    orchestrator.on_fault_detected = on_fault
    
    if orchestrator.start():
        try:
            while True:
                time.sleep(1)
        except KeyboardInterrupt:
            orchestrator.stop()
    else:
        print("Failed to start Hardware Orchestrator.")
        sys.exit(1)

def process_hardfault(args):
    # If input file is provided, try to read registers from it
    cfsr, hfsr, mmfar, bfar, pc = args.cfsr, args.hfsr, args.mmfar, args.bfar, args.pc
    sp, stack_raw = None, None

    if args.input:
        try:
            input_data = {}
            if args.input == "-":
                input_data = json.load(sys.stdin)
            else:
                with open(args.input, "r") as f:
                    input_data = json.load(f)
            
            cfsr = int(str(input_data.get("cfsr", cfsr or 0)), 16)
            hfsr = int(str(input_data.get("hfsr", hfsr or 0)), 16)
            mmfar = int(str(input_data.get("mmfar", mmfar or 0)), 16)
            bfar = int(str(input_data.get("bfar", bfar or 0)), 16)
            pc = int(str(input_data.get("pc", pc or 0)), 16)
            sp = input_data.get("sp")
            stack_raw = input_data.get("stack_raw")
        except Exception as e:
            print(f"Error parsing input JSON: {e}")
            sys.exit(1)

    if cfsr is None or hfsr is None:
        print("Error: CFSR and HFSR values are required.")
        sys.exit(1)

    # Initialize components
    rules_path = os.path.join(os.path.dirname(__file__), "..", "rule_engine", "fault_rules.json")
    decoder = HardFaultDecoder(rules_path)
    engine = DiagnosticEngine()
    
    # Decode registers
    raw_fault_data = decoder.decode(cfsr, hfsr, mmfar, bfar)
    raw_fault_data["sp"] = sp
    raw_fault_data["stack_raw"] = stack_raw
    
    if pc is not None:
        raw_fault_data["crash_location"] = {"address": hex(pc) if isinstance(pc, int) else pc}
    
    # Resolve symbols if ELF is provided
    elf_info = None
    resolver = None
    if args.elf:
        try:
            resolver = ELFResolver(args.elf)
            if pc:
                pc_int = int(str(pc), 16) if isinstance(pc, str) else pc
                elf_info = resolver.resolve_address(pc_int)
        except Exception as e:
            print(f"DEBUG: ELF resolution failed: {e}")

    # Final Diagnostic Analysis
    final_report = engine.analyze(raw_fault_data, elf_info)

    # Resolve symbols for stack trace if possible
    if resolver and final_report.get("stack_trace"):
        for entry in final_report["stack_trace"]:
            try:
                addr_int = int(entry["address"], 16)
                info = resolver.resolve_address(addr_int)
                if info:
                    entry["function"] = info.get("name", "Unknown")
                    entry["file"] = info.get("file", "N/A")
                    entry["line"] = info.get("line", "N/A")
            except:
                pass

    if args.json:
        print(json.dumps(final_report, indent=4))
    else:
        # Confidence Colors
        conf_color = "\033[1;32m" # Green (Deterministic)
        if final_report['confidence'] == "High": conf_color = "\033[1;36m" # Cyan
        elif final_report['confidence'] == "Medium": conf_color = "\033[1;33m" # Yellow
        elif final_report['confidence'] == "Low": conf_color = "\033[1;31m" # Red

        print("\n" + "█"*60)
        print(" HARDCOREAI: PRO-GRADE HARDFAULT DIAGNOSTIC REPORT ")
        print("█"*60)
        
        print(f"\nFAULT TYPE     : \033[1;31m{final_report['fault_type']}\033[0m")
        print(f"CONFIDENCE     : {conf_color}{final_report['confidence']}\033[0m")
        
        print(f"\n\033[1mWHAT HAPPENED\033[0m:")
        print(f"{final_report['what_happened']}")

        if final_report.get("timeline"):
            print(f"\n\033[1mFAULT TIMELINE (RECONSTRUCTED)\033[0m:")
            for note in final_report["timeline"]:
                print(f"  ⚡ {note}")

        print(f"\n\033[1mWHERE\033[0m:")
        w = final_report['where']
        print(f"  • Function : {w['function']}()")
        print(f"  • Location : {w['file']}:{w['line']}")
        print(f"  • Address  : {w['address']}")

        if final_report.get("stack_trace"):
            print(f"\n\033[1mPOTENTIAL ORIGINS (STACK SCAN)\033[0m:")
            for entry in final_report["stack_trace"][:5]: # Top 5
                print(f"  • {entry['address']} -> {entry['function']}() at {entry.get('file', 'N/A')}:{entry.get('line', 'N/A')}")

        print(f"\n\033[1mDIAGNOSTIC EVIDENCE\033[0m:")
        for fact in final_report.get("evidence", []):
            print(f"  ✅ {fact}")

        print(f"\n\033[1mLIKELY SCENARIO\033[0m:")
        print(f"\033[33m{final_report['likely_scenario']}\033[0m")

        print(f"\n\033[1mACTIONABLE FIX\033[0m:")
        for i, step in enumerate(final_report['fix'], 1):
            print(f" {i}. {step}")

        print("\n" + "─"*60)
        print("TECHNICAL DETAILS (BIT-LEVEL):")
        for detail in final_report['raw_details']:
            print(f"  • [\033[36m{detail['register']}\033[0m] {detail['name']}: {detail['description']}")
        print("█"*60 + "\n")

if __name__ == "__main__":
    main()
