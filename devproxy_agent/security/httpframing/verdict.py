from enum import Enum
from dataclasses import dataclass
from typing import Optional

class Verdict(str, Enum):
    ALLOW = "ALLOW"
    REJECT = "REJECT"
    AMBIGUOUS = "AMBIGUOUS"

@dataclass(frozen=True)
class FramingResult:
    verdict: Verdict
    reason: str
    content_length: Optional[int] = None
    has_transfer_encoding: bool = False
    is_chunked: bool = False
