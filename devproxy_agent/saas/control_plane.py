import hashlib
import os
import subprocess
import tempfile
from datetime import datetime, timezone
from typing import Any

import redis.asyncio as redis
import requests
from fastapi import (
    BackgroundTasks,
    Body,
    Depends,
    FastAPI,
    Header,
    HTTPException,
    Response,
    WebSocket,
    WebSocketDisconnect,
)
from pydantic import BaseModel
from redis.exceptions import ConnectionError as RedisConnectionError
from sqlmodel import Session, select

from devproxy_agent.saas.admin import router as admin_router
from devproxy_agent.saas.database import get_session
from devproxy_agent.saas.metrics_router import router as metrics_router
from devproxy_agent.saas.models import AgentRun, Incident, Organization, ProxyNode
from devproxy_agent.saas.payment import router as payment_router
from devproxy_agent.saas.pr_comment_bot import router as pr_comment_bot_router

app = FastAPI(title="Devproxy SaaS Control Plane", version="1.0.0")


# Initialize DB on startup
@app.on_event("startup")
async def on_startup():
    from devproxy_agent.saas.database import init_db

    init_db()


redis_client = redis.from_url(
    os.getenv("REDIS_URL", "redis://localhost:6379/0"), decode_responses=True
)


# Simple WebSocket manager for panel connections
class PanelConnectionManager:
    def __init__(self) -> None:
        self.active_connections: set[WebSocket] = set()

    async def connect(self, websocket: WebSocket) -> None:
        await websocket.accept()
        self.active_connections.add(websocket)

    def disconnect(self, websocket: WebSocket) -> None:
        self.active_connections.discard(websocket)

    async def broadcast(self, message: str) -> None:
        for connection in list(self.active_connections):
            try:
                await connection.send_text(message)
            except Exception:
                self.disconnect(connection)


panel_ws_manager = PanelConnectionManager()


class HeartbeatPayload(BaseModel):
    node_name: str
    version: str


class TelemetryIngestPayload(BaseModel):
    method: str
    path: str
    status_code: int
    redacted_headers: dict[str, Any]
    redacted_request_body: str
    redacted_response_body: str
    latency_ms: float


def hash_api_key(raw_key: str) -> str:
    return hashlib.sha256(raw_key.encode("utf-8")).hexdigest()


async def authenticate_node(
    authorization: str = Header(...),
    session: Session = Depends(get_session),
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


@app.post("/v1/nodes/heartbeat")
async def node_heartbeat(
    payload: HeartbeatPayload,
    auth: tuple[Organization, ProxyNode] = Depends(authenticate_node),
    session: Session = Depends(get_session),
):
    org, node = auth
    node.last_heartbeat_at = datetime.now(timezone.utc)
    node.version = payload.version
    session.add(node)
    session.commit()
    return {
        "status": "ok",
        "routing_config": node.routing_config,
        "quota_remaining": max(0, org.monthly_pr_limit - org.prs_used_this_month),
    }


@app.post("/v1/telemetry/ingest", status_code=202)
async def ingest_error_trace(
    payload: TelemetryIngestPayload,
    background_tasks: BackgroundTasks,
    auth: tuple[Organization, ProxyNode] = Depends(authenticate_node),
    session: Session = Depends(get_session),
):
    org, _ = auth
    error_head = payload.redacted_response_body[:160]
    raw_fp = (
        f"{org.id}:{payload.method}:{payload.path}:{payload.status_code}:{error_head}"
    )
    fingerprint = hashlib.sha256(raw_fp.encode("utf-8")).hexdigest()

    dedup_key = f"dedup:{org.id}:{fingerprint}"
    try:
        dedup_exists = await redis_client.get(dedup_key)
    except RedisConnectionError:
        dedup_exists = None
    if dedup_exists:
        existing = session.exec(
            select(Incident).where(Incident.fingerprint == fingerprint)
        ).first()
        if existing:
            existing.occurrence_count += 1
            existing.last_seen_at = datetime.now(timezone.utc)
            session.add(existing)
            session.commit()
        return {"status": "deduplicated", "fingerprint": fingerprint}

    if org.prs_used_this_month >= org.monthly_pr_limit:
        return {"status": "quota_exceeded", "fingerprint": fingerprint}

    incident = Incident(
        org_id=org.id,
        fingerprint=fingerprint,
        http_method=payload.method,
        route_path=payload.path,
        status_code=payload.status_code,
        error_summary=error_head,
        redacted_request_payload={
            "headers": payload.redacted_headers,
            "body": payload.redacted_request_body,
        },
        redacted_response_body=payload.redacted_response_body,
        latency_ms=payload.latency_ms,
        status="analyzing",
    )

    session.add(incident)
    session.commit()
    try:
        await redis_client.setex(dedup_key, 21600, "1")
    except RedisConnectionError:
        pass
    return {"status": "queued_for_autonomous_loop", "incident_id": incident.id}


@app.get("/v1/panel/incident/{incident_id}")
async def get_incident_detail(
    incident_id: str,
    auth: tuple[Organization, ProxyNode] = Depends(authenticate_node),
    session: Session = Depends(get_session),
):
    incident = session.get(Incident, incident_id)
    if not incident:
        raise HTTPException(status_code=404, detail="Incident not found")
    return {
        "incident_id": incident.id,
        "generated_diff": incident.generated_diff,
        "route_path": incident.route_path,
        "method": getattr(incident, "captured_method", None) or incident.http_method,
    }


@app.get("/v1/metrics/prometheus")
async def prometheus_metrics(
    session: Session = Depends(get_session),
):
    org = session.exec(select(Organization)).first()
    if not org:

        class DummyOrg:
            name = "test"
            monthly_pr_limit = 0
            prs_used_this_month = 0

        org = DummyOrg()
    runs = (
        session.exec(
            select(AgentRun).where(AgentRun.org_id == getattr(org, "id", None))
        ).all()
        if hasattr(org, "id")
        else []
    )
    total_cycles = sum(r.cycles_executed for r in runs)
    succeeded_prs = sum(1 for r in runs if r.status == "succeeded")
    lines = [
        "# HELP devproxy_autonomous_cycles_total Total autonomous LLM cycles executed",
        "# TYPE devproxy_autonomous_cycles_total counter",
        f'devproxy_autonomous_cycles_total{{org="{org.name}"}} {total_cycles}',
        "# HELP devproxy_prs_opened_total Total verified pull requests opened",
        "# TYPE devproxy_prs_opened_total counter",
        f'devproxy_prs_opened_total{{org="{org.name}"}} {succeeded_prs}',
        "# HELP devproxy_monthly_pr_quota_remaining Remaining PR credits in billing cycle",
        "# TYPE devproxy_monthly_pr_quota_remaining gauge",
        f'devproxy_monthly_pr_quota_remaining{{org="{org.name}"}} {max(0, org.monthly_pr_limit - org.prs_used_this_month)}',
    ]
    return Response(content="\n".join(lines) + "\n", media_type="text/plain")


def _execute_git_patch(target_dir: str, diff_text: str) -> tuple[bool, str | None]:
    try:
        subprocess.run(
            ["git", "apply", "-"],
            input=diff_text.encode(),
            cwd=target_dir,
            check=True,
            capture_output=True,
        )
    except subprocess.CalledProcessError as e:
        return False, e.stderr.decode()

    subprocess.run(["git", "add", "."], cwd=target_dir, capture_output=True)
    subprocess.run(
        ["git", "commit", "-m", "Apply fix from DevProxy"],
        cwd=target_dir,
        capture_output=True,
    )
    return True, None


def _replay_incident_request(incident: Incident) -> tuple[bool, int | None, str | None]:
    try:
        req_payload = incident.redacted_request_payload or {}
        headers = getattr(incident, "captured_headers", None)
        if headers is None:
            headers = (
                req_payload.get("headers", {}) if isinstance(req_payload, dict) else {}
            )

        body = getattr(incident, "captured_body", None)
        if body is None:
            body = req_payload.get("body") if isinstance(req_payload, dict) else None

        method = getattr(incident, "captured_method", None) or getattr(
            incident, "http_method", "GET"
        )

        replay_resp = requests.request(
            method=method,
            url=f"http://127.0.0.1:8000{incident.route_path}",
            headers=headers,
            json=body,
            timeout=10,
        )
        return True, replay_resp.status_code, None
    except Exception as exc:
        return False, None, str(exc)


@app.post("/v1/panel/apply_fix")
async def apply_fix(
    payload: dict[str, Any] = Body(...),
    auth: tuple[Organization, ProxyNode] = Depends(authenticate_node),
    session: Session = Depends(get_session),
):
    org, _ = auth
    incident_id = payload.get("incident_id")
    diff_text = payload.get("diff")
    if not incident_id or not diff_text:
        raise HTTPException(status_code=400, detail="incident_id and diff required")

    incident = session.get(Incident, incident_id)
    if not incident:
        raise HTTPException(status_code=404, detail="Incident not found")

    gh_token = None
    if org.github_installation_id:
        from devproxy_agent.saas.github_app import get_installation_access_token

        gh_token = get_installation_access_token(org.github_installation_id)
    repo_url = (
        f"https://x-access-token:{gh_token}@github.com/{org.target_repo_full_name}.git"
        if gh_token
        else None
    )

    with tempfile.TemporaryDirectory(prefix=f"devproxy_apply_{org.id}_") as ws:
        target_dir = ws
        if repo_url:
            subprocess.run(
                ["git", "clone", "--depth", "1", repo_url, target_dir],
                check=True,
                capture_output=True,
            )
        else:
            target_dir = os.getcwd()

        success, err = _execute_git_patch(target_dir, diff_text)
        if not success:
            return {"apply_success": False, "error": err}

        replay_ok, status_code, replay_err = _replay_incident_request(incident)
        if not replay_ok:
            return {"apply_success": True, "replay_success": False, "error": replay_err}

        return {
            "apply_success": True,
            "replay_success": True,
            "replay_status": status_code,
        }


@app.websocket("/ws/panel")
async def websocket_panel_endpoint(websocket: WebSocket) -> None:
    await panel_ws_manager.connect(websocket)
    try:
        while True:
            data = await websocket.receive_text()
            await panel_ws_manager.broadcast(data)
    except WebSocketDisconnect:
        panel_ws_manager.disconnect(websocket)


app.include_router(metrics_router, prefix="/metrics", tags=["metrics"])
app.include_router(pr_comment_bot_router, prefix="/webhooks/pr", tags=["pr_webhook"])
app.include_router(admin_router, prefix="/admin", tags=["admin"])
app.include_router(payment_router, prefix="/payment", tags=["payment"])
