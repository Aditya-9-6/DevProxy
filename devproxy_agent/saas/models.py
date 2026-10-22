from datetime import datetime, timezone
from typing import Optional, Dict, Any
import uuid

from sqlmodel import SQLModel, Field, Column, JSON


def generate_uuid() -> str:
    return str(uuid.uuid4())


class Organization(SQLModel, table=True):
    __tablename__ = "organizations"
    id: str = Field(default_factory=generate_uuid, primary_key=True)
    name: str = Field(index=True)
    billing_email: str = Field(default="")
    plan_tier: str = Field(default="pilot")  # free, pilot, pro, scale
    monthly_pr_limit: int = Field(default=25)
    prs_used_this_month: int = Field(default=0)
    github_installation_id: Optional[int] = Field(default=None)
    target_repo_full_name: Optional[str] = Field(
        default=None
    )  # e.g. "acme-corp/backend-api"
    auto_merge_non_breaking: bool = Field(default=False)
    created_at: datetime = Field(default_factory=lambda: datetime.now(timezone.utc))


class ProxyNode(SQLModel, table=True):
    __tablename__ = "proxy_nodes"
    id: str = Field(default_factory=generate_uuid, primary_key=True)
    org_id: str = Field(foreign_key="organizations.id", index=True)
    node_name: str = Field(default="default-node")  # e.g., "prod-us-east-1"
    api_key_prefix: str = Field(default="dp_live_", index=True)  # e.g., "dp_live_8f92a"
    api_key_hash: str = Field(unique=True, index=True)  # SHA-256 of full token
    version: str = Field(default="0.1.0")
    last_heartbeat_at: Optional[datetime] = Field(default=None)
    routing_config: Dict[str, Any] = Field(default={}, sa_column=Column(JSON))
    is_active: bool = Field(default=True)


class Incident(SQLModel, table=True):
    __tablename__ = "incidents"
    id: str = Field(default_factory=generate_uuid, primary_key=True)
    org_id: str = Field(foreign_key="organizations.id", index=True)
    fingerprint: str = Field(
        index=True
    )  # SHA-256(org_id + method + route + status + error_head)
    http_method: str
    route_path: str
    status_code: int
    error_summary: str
    redacted_request_payload: Dict[str, Any] = Field(default={}, sa_column=Column(JSON))
    captured_method: str = Field(default="")
    captured_headers: Dict[str, Any] = Field(default={}, sa_column=Column(JSON))
    captured_body: Dict[str, Any] = Field(default={}, sa_column=Column(JSON))
    generated_diff: str = Field(default="")
    redacted_response_body: str
    latency_ms: float
    occurrence_count: int = Field(default=1)
    status: str = Field(
        default="queued"
    )  # queued, analyzing, pr_opened, failed, ignored
    created_at: datetime = Field(default_factory=lambda: datetime.now(timezone.utc))
    last_seen_at: datetime = Field(default_factory=lambda: datetime.now(timezone.utc))


class AgentRun(SQLModel, table=True):
    __tablename__ = "agent_runs"
    id: str = Field(default_factory=generate_uuid, primary_key=True)
    org_id: str = Field(foreign_key="organizations.id", index=True)
    incident_id: str = Field(foreign_key="incidents.id", index=True)
    status: str = Field(default="running")  # running, succeeded, failed
    pr_url: Optional[str] = Field(default=None)
    pr_number: Optional[int] = Field(default=None)
    cycles_executed: int = Field(default=1)
    llm_provider_used: Optional[str] = Field(default=None)
    execution_duration_sec: float = Field(default=0.0)
    error_log: Optional[str] = Field(default=None)
    created_at: datetime = Field(default_factory=lambda: datetime.now(timezone.utc))
