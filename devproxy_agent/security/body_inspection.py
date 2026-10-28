from typing import Dict, Optional
from devproxy_agent.security.contentcodec import ContentCodecDecoder
from devproxy_agent.security.decompguard import DecompGuard


class WAFBodyInspector:
    def __init__(self, guard: Optional[DecompGuard] = None):
        self.guard = guard or DecompGuard()
        self.decoder = ContentCodecDecoder(self.guard)

    def inspect_body(self, raw_body: bytes, headers: Dict[str, str]) -> bytes:
        content_encoding = headers.get("content-encoding", "identity")
        codings = self.decoder.parse_codings(content_encoding)
        decoded = self.decoder.decode_stream(raw_body, codings)
        return decoded
