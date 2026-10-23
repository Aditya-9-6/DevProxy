import json
import subprocess
import tempfile
import time
import os
from sqlmodel import Session
from devproxy_agent.saas.database import engine
from devproxy_agent.saas.models import Organization, Incident, AgentRun
from devproxy_agent.saas.github_app import get_installation_access_token


def execute_sandboxed_autonomous_loop(org_id: str, incident_id: str):
    """Execute the autonomous loop in an isolated workspace.
    This function is intended to be called as a background task from FastAPI.
    """
    start_time = time.time()
    with Session(engine) as session:
        org = session.get(Organization, org_id)
        incident = session.get(Incident, incident_id)
        if not org or not incident or not org.target_repo_full_name:
            return

        run_record = AgentRun(org_id=org.id, incident_id=incident.id, status="running")
        session.add(run_record)
        session.commit()
        session.refresh(run_record)

        try:
            # 1. Obtain a short‑lived GitHub App token
            gh_token = get_installation_access_token(org.github_installation_id or 0)
            repo_url = f"https://x-access-token:{gh_token}@github.com/{org.target_repo_full_name}.git"

            # 2. Clone the repository into a temporary workspace
            with tempfile.TemporaryDirectory(prefix=f"devproxy_{org.id}_") as workspace:
                subprocess.run(
                    ["git", "clone", "--depth", "50", repo_url, workspace],
                    check=True,
                    capture_output=True,
                )

                # 3. Write incident context for the autonomous loop
                incident_context_path = os.path.join(
                    workspace, ".devproxy_incident.json"
                )
                with open(incident_context_path, "w") as f:
                    json.dump(
                        {
                            "incident_id": incident.id,
                            "http_method": incident.http_method,
                            "route_path": incident.route_path,
                            "status_code": incident.status_code,
                            "request_payload": incident.redacted_request_payload,
                            "response_body": incident.redacted_response_body,
                            "latency_ms": incident.latency_ms,
                            "instructions": (
                                "Reproduce this exact HTTP failure in a unit test, "
                                "fix the underlying bug without modifying protected auth/env files, "
                                "verify all tests pass, and open a Pull Request."
                            ),
                        },
                        f,
                        indent=2,
                    )

                # 4. Run the agent CLI inside the workspace
                cmd = [
                    "python",
                    "-m",
                    "devproxy_agent.run_agent",
                    "--once",
                    "--workspace",
                    workspace,
                ]
                result = subprocess.run(
                    cmd,
                    capture_output=True,
                    text=True,
                    timeout=600,  # 10‑minute hard cap per cycle
                )

                duration = round(time.time() - start_time, 2)
                if result.returncode == 0:
                    # Extract PR URL from stdout (simple heuristic)
                    pr_url = _extract_pr_url(result.stdout)
                    run_record.status = "succeeded"
                    run_record.pr_url = pr_url
                    run_record.execution_duration_sec = duration
                    incident.status = "pr_opened"
                    org.prs_used_this_month += 1
                else:
                    run_record.status = "failed"
                    run_record.error_log = result.stderr[-2000:]
                    run_record.execution_duration_sec = duration
                    incident.status = "failed"

                session.add(run_record)
                session.add(incident)
                session.add(org)
                session.commit()
        except Exception as exc:
            run_record.status = "failed"
            run_record.error_log = str(exc)
            incident.status = "failed"
            session.add(run_record)
            session.add(incident)
            session.commit()


def _extract_pr_url(stdout: str) -> str:
    """Parse stdout for a GitHub Pull Request URL.
    Returns a placeholder if none found.
    """
    for line in stdout.splitlines():
        if "https://github.com/" in line and "/pull/" in line:
            return line.strip()
    return "PR_CREATED_CHECK_REPO"
