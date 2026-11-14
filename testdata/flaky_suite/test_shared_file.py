"""Shared-file race — often flaky when FlakeHunter uses --parallel."""

import time

from conftest import SHARED_FILE


def _read_counter() -> int:
    if not SHARED_FILE.exists():
        return 0
    raw = SHARED_FILE.read_text(encoding="utf-8").strip()
    if not raw:
        return 0
    return int(raw)


def test_shared_counter_increment():
    current = _read_counter()
    nxt = current + 1
    SHARED_FILE.write_text(str(nxt), encoding="utf-8")
    time.sleep(0.015)  # widen the race window under parallel workers
    got = _read_counter()
    # Fails if another concurrent run overwrote the file between write and read.
    assert got == nxt, f"expected {nxt}, got {got}"
