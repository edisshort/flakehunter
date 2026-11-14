"""Control: should always fail (not flaky)."""


def test_always_fails():
    assert False, "intentional consistent failure"
