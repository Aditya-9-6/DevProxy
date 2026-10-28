import sqlite3
import datetime
from pathlib import Path
from devproxy_agent.config import settings

DB_PATH = Path(settings.CYCLE_LOG_DB)


def init_db():
    """Initialize the SQLite database and ensure the cycles table exists."""
    conn = sqlite3.connect(DB_PATH)
    try:
        conn.execute(
            """
            CREATE TABLE IF NOT EXISTS cycles (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                ts TIMESTAMP NOT NULL
            )
            """
        )
        conn.commit()
    finally:
        conn.close()


def log_cycle():
    """Record a processing cycle with the current timestamp."""
    init_db()
    conn = sqlite3.connect(DB_PATH)
    try:
        now = datetime.datetime.utcnow().isoformat(timespec="seconds")
        conn.execute("INSERT INTO cycles (ts) VALUES (?)", (now,))
        conn.commit()
    finally:
        conn.close()


def count_cycles_last(hours: int = 24) -> int:
    """Return the number of cycles logged in the past *hours* hours."""
    init_db()
    conn = sqlite3.connect(DB_PATH)
    try:
        cutoff = datetime.datetime.utcnow() - datetime.timedelta(hours=hours)
        cur = conn.execute(
            "SELECT COUNT(*) FROM cycles WHERE ts >= ?",
            (cutoff.isoformat(timespec="seconds"),),
        )
        return cur.fetchone()[0]
    finally:
        conn.close()


def estimate_prs_per_day() -> float:
    """Estimate how many PRs can be merged in a 24‑hour window.

    Uses the configured PR_PER_CYCLE_FACTOR multiplied by the number of
    cycles observed in the last 24 hours.
    """
    cycles = count_cycles_last(24)
    factor = getattr(settings, "PR_PER_CYCLE_FACTOR", 0.5)
    return cycles * factor
