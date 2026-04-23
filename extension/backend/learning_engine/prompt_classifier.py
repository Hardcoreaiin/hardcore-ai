import re
import logging

logger = logging.getLogger(__name__)

class PromptClassifier:
    """
    Classifies the user prompt to determine the engineering mode:
    FIRMWARE_MODE, EMBEDDED_DESIGN_MODE, or SYSTEM_ARCHITECTURE_MODE.
    """
    
    FIRMWARE_MODE = "FIRMWARE_MODE"
    EMBEDDED_DESIGN_MODE = "EMBEDDED_DESIGN_MODE"
    SYSTEM_ARCHITECTURE_MODE = "SYSTEM_ARCHITECTURE_MODE"
    
    # Keywords indicating a simple firmware task
    FIRMWARE_KEYWORDS = [
        "blink", "led", "read sensor", "uart", "pwm", "i2c", "spi", "adc", 
        "dac", "timer", "interrupt", "delay", "print", "serial", "temperature",
        "humidity", "dht", "bme280", "mpu6050", "motor control"
    ]
    
    # Keywords indicating a board-level embedded design
    EMBEDDED_DESIGN_KEYWORDS = [
        "build", "robot", "sensor node", "weather station", "drone", 
        "data logger", "wearable", "smart home", "esp32", "stm32", "arduino"
    ]
    
    # Keywords indicating a complex system architecture / MPU
    SYSTEM_ARCHITECTURE_KEYWORDS = [
        "secure", "imx", "sitara", "linux", "gateway", "edge device", 
        "industrial", "som", "mpu", "cortex-a", "yocto", "boot chain",
        "compliance", "router", "vision system"
    ]
    
    @classmethod
    def classify_prompt(cls, user_prompt: str) -> str:
        prompt_lower = user_prompt.lower()
        
        # 1. Check for system architecture keywords first (highest precedence)
        for keyword in cls.SYSTEM_ARCHITECTURE_KEYWORDS:
            if re.search(r'\b' + re.escape(keyword) + r'\b', prompt_lower):
                logger.info(f"Classified as SYSTEM_ARCHITECTURE_MODE (matched '{keyword}')")
                return cls.SYSTEM_ARCHITECTURE_MODE
                
        # 2. Check for embedded design keywords
        for keyword in cls.EMBEDDED_DESIGN_KEYWORDS:
            if re.search(r'\b' + re.escape(keyword) + r'\b', prompt_lower):
                logger.info(f"Classified as EMBEDDED_DESIGN_MODE (matched '{keyword}')")
                return cls.EMBEDDED_DESIGN_MODE
                
        # 3. Check for firmware keywords
        for keyword in cls.FIRMWARE_KEYWORDS:
            if re.search(r'\b' + re.escape(keyword) + r'\b', prompt_lower):
                logger.info(f"Classified as FIRMWARE_MODE (matched '{keyword}')")
                return cls.FIRMWARE_MODE
                
        # 4. Fallback logic based on length/complexity
        if len(user_prompt.split()) > 20:
            logger.info("Classified as EMBEDDED_DESIGN_MODE (fallback based on complexity)")
            return cls.EMBEDDED_DESIGN_MODE
            
        logger.info("Classified as FIRMWARE_MODE (default fallback)")
        return cls.FIRMWARE_MODE
