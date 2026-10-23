from devproxy_agent.security.httpframing.parser import HTTPFramingParser
from devproxy_agent.security.httpframing.verdict import FramingResult


class HTTP1RequestGuard:
    @staticmethod
    def inspect_request(header_bytes: bytes) -> FramingResult:
        return HTTPFramingParser.parse_headers_and_frame(header_bytes)
