from devproxy_agent.security.httpframing.parser import HTTPFramingParser
from devproxy_agent.security.httpframing.verdict import Verdict


def test_parser_normal_get():
    raw = b"GET /index.html HTTP/1.1\r\nHost: example.com\r\n\r\n"
    res = HTTPFramingParser.parse_headers_and_frame(raw)
    assert res.verdict == Verdict.ALLOW
    assert res.content_length is None
    assert not res.has_transfer_encoding


def test_parser_normal_post_cl():
    raw = b"POST /api HTTP/1.1\r\nHost: example.com\r\nContent-Length: 10\r\n\r\n"
    res = HTTPFramingParser.parse_headers_and_frame(raw)
    assert res.verdict == Verdict.ALLOW
    assert res.content_length == 10


def test_parser_cl_te_smuggling():
    raw = b"POST /api HTTP/1.1\r\nHost: example.com\r\nContent-Length: 5\r\nTransfer-Encoding: chunked\r\n\r\n"
    res = HTTPFramingParser.parse_headers_and_frame(raw)
    assert res.verdict == Verdict.AMBIGUOUS
    assert res.reason == "cl_te_ambiguity"


def test_parser_multiple_host():
    raw = b"GET / HTTP/1.1\r\nHost: a.com\r\nHost: b.com\r\n\r\n"
    res = HTTPFramingParser.parse_headers_and_frame(raw)
    assert res.verdict == Verdict.REJECT
    assert res.reason == "multiple_host_headers"


def test_parser_obsolete_folding():
    raw = b"GET / HTTP/1.1\r\nHost: example.com\r\nX-Header: foo\r\n bar\r\n\r\n"
    res = HTTPFramingParser.parse_headers_and_frame(raw)
    assert res.verdict == Verdict.REJECT
    assert res.reason == "obsolete_line_folding"
