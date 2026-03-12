"""
ESP32 Microcontroller Abstraction Layer (MCAL) Definitions
"""

from typing import List, Dict, Any

class ESP32Def:
    """ESP32 Hardware Definitions."""
    
    # Pin capabilities
    AVAILABLE_GPIO = list(range(0, 40))
    
    # Input-only pins (no internal pullups/pulldowns)
    INPUT_ONLY_PINS = [34, 35, 36, 37, 38, 39]
    
    # Strapping pins (careful during boot)
    STRAPPING_PINS = [0, 2, 5, 12, 15]
    
    # Flash pins (connected to internal SPI flash, DO NOT USE)
    FLASH_PINS = [6, 7, 8, 9, 10, 11]
    
    # PWM capable pins (almost all output pins, but good to list preferences)
    PWM_CAPABLE = [
        2, 4, 5, 12, 13, 14, 15, 16, 17, 18, 19, 21, 22, 23, 
        25, 26, 27, 32, 33
    ]
    
    # ADC1 (safe to use with WiFi)
    ADC1_PINS = [32, 33, 34, 35, 36, 39]
    
    # ADC2 (restricted when WiFi is on)
    ADC2_PINS = [0, 2, 4, 12, 13, 14, 15, 25, 26, 27]
    
    # Default I2C
    I2C_DEFAULT = {"SDA": 21, "SCL": 22}
    
    # Default SPI
    SPI_DEFAULT = {"MOSI": 23, "MISO": 19, "SCK": 18, "CS": 5}
    
    # Reserved pins (Boot/Flash/UART0)
    RESERVED_PINS = [0, 1, 3] + FLASH_PINS

    @staticmethod
    def is_valid_gpio(pin: int) -> bool:
        """Check if pin is a valid GPIO number."""
        return pin in ESP32Def.AVAILABLE_GPIO and pin not in ESP32Def.FLASH_PINS

    @staticmethod
    def is_output_capable(pin: int) -> bool:
        """Check if pin can be an output."""
        return ESP32Def.is_valid_gpio(pin) and pin not in ESP32Def.INPUT_ONLY_PINS
