from fastapi import APIRouter
from devproxy_agent.metrics import count_cycles_last, estimate_prs_per_day

router = APIRouter()


@router.get("/metrics")
async def get_metrics():
    """Return basic metrics: cycles in last 24 h and estimated PRs per day."""
    cycles = count_cycles_last(86400)  # 24 h
    pr_estimate = estimate_prs_per_day()
    return {"cycles_last_24h": cycles, "pr_estimate_per_day": pr_estimate}
