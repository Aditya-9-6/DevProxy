import gzip
import zlib
import sys
import pathlib
import pytest

sys.path.append(str(pathlib.Path(__file__).resolve().parents[1]))

from devproxy_agent.security.decompguard import DecompGuard, DecompPolicy
from devproxy_agent.security.contentcodec import ContentCodecDecoder
from devproxy_agent.security.body_inspection import WAFBodyInspector


def test_parse_codings():
    decoder = ContentCodecDecoder()
    assert decoder.parse_codings("gzip, deflate") == ["gzip", "deflate"]
    assert decoder.parse_codings("identity") == ["identity"]
    with pytest.raises(ValueError):
        decoder.parse_codings("unknown-codec")


def test_gzip_decoding_success():
    original = b"Hello GossipMesh WAF Pipeline!"
    compressed = gzip.compress(original)
    inspector = WAFBodyInspector()
    decoded = inspector.inspect_body(compressed, {"content-encoding": "gzip"})
    assert decoded == original


def test_deflate_decoding_success():
    original = b"Hello Deflate Stream!"
    compressed = zlib.compress(original)
    inspector = WAFBodyInspector()
    decoded = inspector.inspect_body(compressed, {"content-encoding": "deflate"})
    assert decoded == original


def test_decompression_bomb_guard():
    policy = DecompPolicy(
        max_compressed_bytes=1000,
        max_expanded_bytes=500,
        max_expansion_ratio=5.0,
        max_nesting_depth=3,
        max_work_units=10000,
    )
    guard = DecompGuard(policy)
    inspector = WAFBodyInspector(guard)

    # Generate payload that exceeds expanded byte limit
    large_original = b"A" * 600
    compressed = gzip.compress(large_original)

    with pytest.raises(ValueError, match="Expanded bytes exceeded limit"):
        inspector.inspect_body(compressed, {"content-encoding": "gzip"})
