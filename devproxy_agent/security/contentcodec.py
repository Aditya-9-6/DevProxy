import gzip
import zlib
from typing import List, Optional
from devproxy_agent.security.decompguard import DecompGuard


class ContentCodecDecoder:
    def __init__(self, guard: Optional[DecompGuard] = None):
        self.guard = guard or DecompGuard()

    def parse_codings(self, accept_encoding_header: str) -> List[str]:
        if not accept_encoding_header:
            return ["identity"]
        codings = [c.strip().lower() for c in accept_encoding_header.split(",")]
        valid_codings = []
        for c in codings:
            if not c:
                continue
            if c not in ("identity", "gzip", "deflate", "compress"):
                raise ValueError(f"Unsupported or malformed coding token: {c}")
            valid_codings.append(c)
        return valid_codings or ["identity"]

    def decode_stream(self, stream_data: bytes, codings: List[str]) -> bytes:
        compressed_size = len(stream_data)
        current_data = stream_data
        nesting = len(codings)

        for idx, coding in enumerate(reversed(codings)):
            if coding == "identity":
                continue
            elif coding == "gzip":
                try:
                    current_data = gzip.decompress(current_data)
                except Exception as e:
                    raise ValueError(f"Malformed gzip payload: {e}")
            elif coding == "deflate":
                try:
                    current_data = zlib.decompress(current_data)
                except Exception as e:
                    raise ValueError(f"Malformed deflate payload: {e}")
            else:
                raise ValueError(f"Unsupported coding: {coding}")

            expanded_size = len(current_data)
            work_units = compressed_size + expanded_size
            self.guard.check_limits(
                compressed_bytes=compressed_size,
                expanded_bytes=expanded_size,
                nesting_depth=nesting - idx,
                work_units=work_units,
            )

        return current_data
