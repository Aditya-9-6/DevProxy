from dataclasses import dataclass
from enum import Enum, auto


class Verdict(Enum):
    ALLOW = auto()
    REJECT = auto()
    AMBIGUOUS = auto()


@dataclass(frozen=True)
class FramingResult:
    verdict: Verdict
    reason: str
    content_length: int | None = None
    has_transfer_encoding: bool = False
    is_chunked: bool = False
