import sys
import pathlib
import pytest

sys.path.append(str(pathlib.Path(__file__).resolve().parents[1]))

from devproxy_agent.security.http1framing import (
    HTTP1FramingDetector,
    FramingAmbiguityError,
)


def test_valid_headers():
    headers = {
        "Host": "example.com",
        "Content-Length": "123",
    }
    findings = HTTP1FramingDetector.validate_headers(headers)
    assert len(findings) == 0


def test_duplicate_content_length_conflict():
    conflict_headers = {
        "Host": "example.com",
        "Content-Length": "10",
        "Transfer-Encoding": "chunked",
    }
    findings = HTTP1FramingDetector.validate_headers(conflict_headers)
    assert "TRANSFER_ENCODING_AND_CONTENT_LENGTH_CONFLICT" in findings

    with pytest.raises(FramingAmbiguityError):
        HTTP1FramingDetector.assert_secure_framing(conflict_headers)


def test_invalid_content_length():
    bad_headers = {
        "Host": "example.com",
        "Content-Length": "abc",
    }
    findings = HTTP1FramingDetector.validate_headers(bad_headers)
    assert "INVALID_CONTENT_LENGTH_FORMAT" in findings
