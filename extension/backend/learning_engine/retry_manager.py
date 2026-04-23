import time
import logging
import asyncio
from typing import Callable, Any

logger = logging.getLogger(__name__)

class RetryManager:
    """
    Manages retry logic with exponential backoff for external API calls,
    specifically designed to handle LLM '503 Overloaded' and '429 Too Many Requests' errors.
    """
    
    def __init__(self, max_retries: int = 5, base_delay: float = 2.0, max_timeout: float = 180.0):
        self.max_retries = max_retries
        self.base_delay = base_delay
        self.max_timeout = max_timeout
        
    async def execute_with_retry(self, func: Callable, *args, **kwargs) -> Any:
        """
        Executes an async function with exponential backoff retry logic.
        """
        start_time = time.time()
        attempt = 0
        
        while attempt <= self.max_retries:
            try:
                # Check absolute timeout
                if time.time() - start_time > self.max_timeout:
                    logger.error(f"RetryManager: Exceeded maximum timeout of {self.max_timeout}s.")
                    raise TimeoutError(f"Operation timed out after {self.max_timeout}s.")
                    
                return await func(*args, **kwargs)
                
            except Exception as e:
                error_msg = str(e).lower()
                attempt += 1
                
                # Check if it's a retriable error (e.g., 503, 429, overloaded)
                is_retriable = any(kw in error_msg for kw in ["503", "429", "overloaded", "quota", "rate limit", "connection reset", "timeout"])
                
                if attempt > self.max_retries or not is_retriable:
                    logger.error(f"RetryManager: Failed after {attempt} attempts. Error: {e}")
                    raise e
                    
                # Calculate delay with exponential backoff and a bit of jitter
                delay = min(self.base_delay * (2 ** (attempt - 1)), 15.0)
                
                logger.warning(f"RetryManager: Attempt {attempt}/{self.max_retries} failed ({e}). Retrying in {delay:.1f}s...")
                await asyncio.sleep(delay)
                
        raise Exception("RetryManager: Reached unreachable code block.")
