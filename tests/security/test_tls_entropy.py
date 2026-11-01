from gossipmesh.security.tls_entropy import calculate_shannon_entropy


def test_calculate_shannon_entropy() -> None:
    assert calculate_shannon_entropy(b"") == 0.0
    uniform_entropy = calculate_shannon_entropy(b"\x01\x02\x03\x04")
    assert uniform_entropy > 0.0
