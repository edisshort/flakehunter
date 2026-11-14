"""Order-sensitive pair: B fails if A has not left a flag in this process."""

from conftest import ORDER_FLAG


def test_order_setup_a():
    ORDER_FLAG.write_text("ready", encoding="utf-8")
    assert ORDER_FLAG.read_text(encoding="utf-8") == "ready"


def test_order_depends_on_a():
    # In a fresh process without A running first, this fails.
    # When pytest collects both in one session and A runs first, it passes.
    assert ORDER_FLAG.exists(), "order flag missing — ran before setup or clean process"
    assert ORDER_FLAG.read_text(encoding="utf-8") == "ready"
