"""Timing-sensitive assertion — occasional race against a short sleep."""

import time
import random


def test_timing_window():
    start = time.perf_counter()
    time.sleep(0.01 + random.random() * 0.04)
    elapsed = time.perf_counter() - start
    # Brittle threshold: sometimes the sleep overshoots on a busy machine.
    assert elapsed < 0.035, f"too slow: {elapsed:.4f}s"
