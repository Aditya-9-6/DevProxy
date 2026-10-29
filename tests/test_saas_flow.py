import pytest
import sys
import pathlib

sys.path.append(str(pathlib.Path(__file__).resolve().parents[1]))
from fastapi.testclient import TestClient
from devproxy_agent.saas.control_plane import app
from devproxy_agent.saas.database import init_db


@pytest.fixture(autouse=True)
def setup_database():
    # Reset DB for each test run
    init_db()
    yield


client = TestClient(app)


def test_full_saas_workflow():
    # 1️⃣ Organization creation
    resp = client.post("/admin/orgs", params={"name": "Pilot Test Corp"})
    assert resp.status_code == 200
    data = resp.json()
    assert "token" in data and data["token"].startswith("dp_live_")
    token = data["token"]
    headers = {"Authorization": f"Bearer {token}"}

    # 2️⃣ Node heartbeat
    hb = client.post(
        "/v1/nodes/heartbeat",
        headers=headers,
        json={"node_name": "ci-runner-node", "version": "0.1.0"},
    )
    assert hb.status_code == 200
    hb_data = hb.json()
    assert hb_data["status"] == "ok"
    assert "quota_remaining" in hb_data

    # 3️⃣ Telemetry ingest (simulated 500 error)
    tele = client.post(
        "/v1/telemetry/ingest",
        headers=headers,
        json={
            "method": "POST",
            "path": "/api/v1/checkout",
            "status_code": 500,
            "error_snippet": "TypeError: test failure",
            "redacted_headers": {},
            "redacted_request_body": '{"test": true}',
            "redacted_response_body": "",
            "latency_ms": 120,
        },
    )
    # 202 Accepted – queued for autonomous loop or deduplicated
    assert tele.status_code == 202
    tel_data = tele.json()
    assert tel_data["status"] in ["queued_for_autonomous_loop", "deduplicated"]
    incident_id = tel_data.get("incident_id")

    # 4️⃣ PR‑comment endpoint (uses incident_id)
    pr = client.post(
        "/webhooks/pr/v1/pr_comment",
        headers=headers,
        json={
            "incident_id": incident_id,
            "pr_url": "https://github.com/example/repo/pull/1",
        },
    )
    assert pr.status_code == 200
    pr_data = pr.json()
    assert "comment" in pr_data
    assert "DevProxy Fix Summary" in pr_data["comment"]

    # 5️⃣ Prometheus metrics endpoint
    metrics = client.get("/v1/metrics/prometheus", headers=headers)
    assert metrics.status_code == 200
    assert "devproxy_autonomous_cycles_total" in metrics.text
