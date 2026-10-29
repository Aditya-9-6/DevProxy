import hashlib
from fastapi import Header, HTTPException, Depends
from sqlmodel import select
from devproxy_agent.saas.models import Organization, ProxyNode
from devproxy_agent.saas.database import get_session


def hash_api_key(raw_key: str) -> str:
    return hashlib.sha256(raw_key.encode("utf-8")).hexdigest()


async def authenticate_node(
    authorization: str = Header(...),
    session=Depends(get_session),
) -> tuple[Organization, ProxyNode]:
    if not authorization.startswith("Bearer dp_live_"):
        raise HTTPException(status_code=401, detail="Invalid Devproxy token format")
    raw_token = authorization.replace("Bearer ", "").strip()
    token_hash = hash_api_key(raw_token)
    node = session.exec(
        select(ProxyNode).where(ProxyNode.api_key_hash == token_hash)
    ).first()
    if not node or not node.is_active:
        raise HTTPException(
            status_code=401, detail="Unauthorized or inactive proxy node"
        )
    org = session.get(Organization, node.org_id)
    if not org:
        raise HTTPException(status_code=401, detail="Organization not found")
    return org, node
