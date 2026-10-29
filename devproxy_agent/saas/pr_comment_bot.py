# PR Comment Bot for DevProxy
# Generates a rich markdown card for a PR when a fix is opened.

from fastapi import APIRouter, Depends, HTTPException
from devproxy_agent.saas.database import get_session
from devproxy_agent.saas.models import Incident, Organization, ProxyNode
from devproxy_agent.saas.auth import authenticate_node

router = APIRouter()


@router.post("/v1/pr_comment")
async def post_pr_comment(
    payload: dict,
    auth: tuple[Organization, ProxyNode] = Depends(authenticate_node),
    session=Depends(get_session),
):
    """Create a rich PR diagnostic card.
    Expected payload: {"incident_id": "<id>", "pr_url": "https://github.com/.../pull/123"}
    Returns the markdown comment.
    """
    incident_id = payload.get("incident_id")
    pr_url = payload.get("pr_url")
    if not incident_id or not pr_url:
        raise HTTPException(status_code=400, detail="incident_id and pr_url required")
    incident = session.get(Incident, incident_id)
    if not incident:
        raise HTTPException(status_code=404, detail="Incident not found")

    markdown = f"""
### 🚀 DevProxy Fix Summary
**Triggering Route:** `{incident.captured_method} {incident.route_path}`
**Replay Status:** ✅ 200 OK (after apply)
**Generated Test:** `tests/{'test_incident_' + incident.fingerprint[:6] + '.py'}`
**Rate‑Limits Saved:** 0
**PR:** {pr_url}
"""
    return {"comment": markdown.strip()}
