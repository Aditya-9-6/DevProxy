import threading
from dataclasses import dataclass
from typing import Optional


@dataclass(frozen=True)
class DecompPolicy:
    max_compressed_bytes: int = 10 * 1024 * 1024
    max_expanded_bytes: int = 50 * 1024 * 1024
    max_expansion_ratio: float = 100.0
    max_nesting_depth: int = 3
    max_work_units: int = 1000000


class DecompGuard:
    def __init__(self, policy: Optional[DecompPolicy] = None):
        self._policy = policy or DecompPolicy()
        self._lock = threading.Lock()

    @property
    def policy(self) -> DecompPolicy:
        with self._lock:
            return self._policy

    def update_policy(self, new_policy: DecompPolicy):
        with self._lock:
            self._policy = new_policy

    def check_limits(
        self,
        compressed_bytes: int,
        expanded_bytes: int,
        nesting_depth: int,
        work_units: int,
    ):
        p = self.policy
        if compressed_bytes > p.max_compressed_bytes:
            raise ValueError(
                f"Compressed bytes exceeded limit: {compressed_bytes} > {p.max_compressed_bytes}"
            )
        if expanded_bytes > p.max_expanded_bytes:
            raise ValueError(
                f"Expanded bytes exceeded limit: {expanded_bytes} > {p.max_expanded_bytes}"
            )
        if compressed_bytes > 0:
            ratio = expanded_bytes / float(compressed_bytes)
            if ratio > p.max_expansion_ratio:
                raise ValueError(
                    f"Expansion ratio exceeded limit: {ratio} > {p.max_expansion_ratio}"
                )
        if nesting_depth > p.max_nesting_depth:
            raise ValueError(
                f"Nesting depth exceeded limit: {nesting_depth} > {p.max_nesting_depth}"
            )
        if work_units > p.max_work_units:
            raise ValueError(
                f"Work units exceeded limit: {work_units} > {p.max_work_units}"
            )
