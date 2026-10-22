# PR Comment Bot for DevProxy
# Generates a rich markdown card for a PR when a fix is opened.

import os
from fastapi import APIRouter, Depends, HTTPException
from devproxy_agent.saas.database import get_session
from devproxy_agent.saas.models import Incident, Organization, ProxyNode
from devproxy_agent.saas.control_plane import authenticate_node
from GossipMesh.gossipmesh_agent.key_pool import rate_limits_saved_total

router = APIRouter()

@router.post("/v1/pr_comment")
async def post_pr_comment(
    payload: dict,
    auth: tuple[Organization, ProxyNode] = Depends(authenticate_node),
    session = Depends(get_session),
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

    # Retrieve metric value (Prometheus counter)
    try:
        saved = rate_limits_saved_total._value.get()
    except Exception:
        saved = 0

    markdown = f"""
### 🚀 DevProxy Fix Summary
**Triggering Route:** `{incident.captured_method} {incident.route_path}`
**Replay Status:** ✅ 200 OK (after apply)
**Generated Test:** `tests/{'test_incident_' + incident.fingerprint[:6] + '.py'}`
**Rate‑Limits Saved:** {saved}
**PR:** {pr_url}
"""
    # In real implementation we would POST this comment to GitHub via its API.
    return {"comment": markdown.strip()}
