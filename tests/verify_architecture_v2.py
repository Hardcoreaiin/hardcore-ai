import asyncio
import os
import sys
import json

# Add parent directory to sys.path to import from learning_engine
sys.path.append(os.path.abspath(os.path.join(os.path.dirname(__file__), "..")))

from learning_engine.architecture_ai import ArchitectureAI

async def run_tests():
    ai = ArchitectureAI()
    
    print("--- Test 1: MCU Detection (ESP32) ---")
    mcu_type = await ai._classify_processor("I want to build a weather station with ESP32 and DHT22")
    print(f"Detected: {mcu_type}")
    assert mcu_type == "MCU"

    print("\n--- Test 2: MPU Detection (i.MX93) ---")
    mpu_type = await ai._classify_processor("Design a high-end gateway using NXP i.MX93 with 8GB RAM")
    print(f"Detected: {mpu_type}")
    assert mpu_type == "MPU"

    print("\n--- Test 3: MPU Firmware Suppression ---")
    firmware_res = await ai.generate_firmware("Design a system with i.MX93")
    print(f"Message: {firmware_res.get('message')}")
    assert "MPU-class system detected" in firmware_res.get('message')
    assert "architecture_notice.md" in firmware_res.get('files')

    print("\n--- Test 4: MPU Firmware Explicit Core Request ---")
    firmware_res_core = await ai.generate_firmware("Give me the code for the Cortex-M33 core on an i.MX93 project")
    print(f"Message: {firmware_res_core.get('message')}")
    assert "Industrial firmware package" in firmware_res_core.get('message')
    assert "main.cpp" in firmware_res_core.get('files')

    print("\n--- Test 5: Multi-File Output Consistency ---")
    firmware_res_mcu = await ai.generate_firmware("ESP32 blink code")
    print(f"Files: {list(firmware_res_mcu.get('files', {}).keys())}")
    assert "main.cpp" in firmware_res_mcu.get('files')
    assert "config.h" in firmware_res_mcu.get('files')

    print("\n✅ All Backend Tests Passed!")

if __name__ == "__main__":
    asyncio.run(run_tests())
