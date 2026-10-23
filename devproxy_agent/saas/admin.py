from fastapi import APIRouter, Depends
from sqlmodel import Session
from devproxy_agent.saas.models import Organization, ProxyNode
from devproxy_agent.saas.database import get_session
import secrets
import hashlib

router = APIRouter()


@router.post("/orgs")
async def create_organization(name: str, session: Session = Depends(get_session)):
    # Create organization
    org = Organization(name=name, monthly_pr_limit=1000, prs_used_this_month=0)
    session.add(org)
    session.commit()
    session.refresh(org)
    # Create a proxy node with a token
    raw_token = "dp_live_" + secrets.token_urlsafe(24)
    token_hash = hashlib.sha256(raw_token.encode("utf-8")).hexdigest()
    node = ProxyNode(
        org_id=org.id,
        api_key_hash=token_hash,
        is_active=True,
        routing_config={},
        last_heartbeat_at=None,
        version="devproxy-0.1",
    )
    session.add(node)
    session.commit()
    session.refresh(node)
    return {"org_id": org.id, "node_id": node.id, "token": raw_token}
