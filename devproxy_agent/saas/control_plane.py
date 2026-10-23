import os
import hashlib
import subprocess
import tempfile
import requests
from datetime import datetime, timezone
from redis.exceptions import ConnectionError as RedisConnectionError
from fastapi import (
    FastAPI,
    Depends,
    Header,
    HTTPException,
    BackgroundTasks,
    Response,
    WebSocket,
    WebSocketDisconnect,
    Body,
)
from pydantic import BaseModel
import redis.asyncio as redis
from sqlmodel import select

from devproxy_agent.saas.models import Organization, ProxyNode, Incident, AgentRun
from devproxy_agent.saas.database import get_session

from devproxy_agent.saas.metrics_router import router as metrics_router
from devproxy_agent.saas.pr_comment_bot import router as pr_comment_bot_router
from devproxy_agent.saas.admin import router as admin_router
from devproxy_agent.saas.payment import router as payment_router

app = FastAPI(title="Devproxy SaaS Control Plane", version="1.0.0")


@app.on_event("startup")
async def on_startup():
    from devproxy_agent.saas.database import init_db

    init_db()


redis_client = redis.from_url(
    os.getenv("REDIS_URL", "redis://localhost:6379/0"), decode_responses=True
)


class PanelConnectionManager:
    def __init__(self):
        self.active_connections: set[WebSocket] = set()

    async def connect(self, websocket: WebSocket):
        await websocket.accept()
        self.active_connections.add(websocket)

    def disconnect(self, websocket: WebSocket):
        self.active_connections.discard(websocket)

    async def broadcast(self, message: str):
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
    redacted_headers: dict
    redacted_request_body: str
    redacted_response_body: str
    latency_ms: float


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


@app.post("/v1/nodes/heartbeat")
async def node_heartbeat(
    payload: HeartbeatPayload,
    auth: tuple[Organization, ProxyNode] = Depends(authenticate_node),
    session=Depends(get_session),
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
    session=Depends(get_session),
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
        captured_method=payload.method,
        captured_headers=payload.redacted_headers,
        captured_body=payload.redacted_request_body,
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
    session=Depends(get_session),
):
    incident = session.get(Incident, incident_id)
    if not incident:
        raise HTTPException(status_code=404, detail="Incident not found")
    return {
        "incident_id": incident.id,
        "generated_diff": incident.generated_diff,
        "route_path": incident.route_path,
        "method": incident.captured_method,
    }


@app.get("/v1/metrics/prometheus")
async def prometheus_metrics(
    session=Depends(get_session),
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


@app.post("/v1/panel/apply_fix")
async def apply_fix(
    payload: dict = Body(...),
    auth: tuple[Organization, ProxyNode] = Depends(authenticate_node),
    session=Depends(get_session),
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
        if repo_url:
            subprocess.run(
                ["git", "clone", "--depth", "1", repo_url, ws],
                check=True,
                capture_output=True,
            )
        else:
            ws = os.getcwd()
        try:
            subprocess.run(
                ["git", "apply", "-"],
                input=diff_text.encode(),
                cwd=ws,
                check=True,
                capture_output=True,
            )
        except subprocess.CalledProcessError as e:
            return {"apply_success": False, "error": e.stderr.decode()}

        subprocess.run(["git", "add", "."], cwd=ws, capture_output=True)
        subprocess.run(
            ["git", "commit", "-m", "Apply fix from DevProxy"],
            cwd=ws,
            capture_output=True,
        )

        try:
            replay_resp = requests.request(
                method=incident.captured_method,
                url=f"http://127.0.0.1:8000{incident.route_path}",
                headers=incident.captured_headers,
                json=incident.captured_body,
                timeout=10,
            )
            replay_status = replay_resp.status_code
        except Exception as exc:
            return {"apply_success": True, "replay_success": False, "error": str(exc)}

        return {
            "apply_success": True,
            "replay_success": True,
            "replay_status": replay_status,
        }


@app.websocket("/ws/panel")
async def websocket_panel_endpoint(websocket: WebSocket):
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
