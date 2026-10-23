from devproxy_agent.proxy.http1.request_guard import HTTP1RequestGuard
from devproxy_agent.security.httpframing.verdict import Verdict


def test_request_guard_allow():
    raw = b"GET /health HTTP/1.1\r\nHost: localhost\r\n\r\n"
    res = HTTP1RequestGuard.inspect_request(raw)
    assert res.verdict == Verdict.ALLOW


def test_request_guard_reject_smuggling():
    raw = b"POST /submit HTTP/1.1\r\nHost: localhost\r\nContent-Length: 10\r\nTransfer-Encoding: chunked\r\n\r\n"
    res = HTTP1RequestGuard.inspect_request(raw)
    assert res.verdict == Verdict.AMBIGUOUS
