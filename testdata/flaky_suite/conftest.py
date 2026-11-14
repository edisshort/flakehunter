"""Shared fixtures for the intentional flaky demo suite."""

from pathlib import Path

# Suite-local shared paths (intentionally not process-isolated).
SUITE_DIR = Path(__file__).resolve().parent
SHARED_FILE = SUITE_DIR / "_shared_counter.txt"
ORDER_FLAG = SUITE_DIR / "_order_flag.txt"
