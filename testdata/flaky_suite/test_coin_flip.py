"""Probabilistic flake: fails ~30% of the time with no code change."""

import random


def test_coin_flip():
    # Fixed seed is NOT used — intentional non-determinism for demo.
    if random.random() < 0.30:
        assert False, "coin landed on fail"
