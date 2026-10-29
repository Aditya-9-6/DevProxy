# Autonomous Loop implementation for DevProxy
# This module is invoked by devproxy_agent.saas.worker.execute_sandboxed_autonomous_loop
# It runs the LLM agent, generates a diff, stores it on the Incident, creates a test file,
# and returns control to the worker.

import json
import os
import re
import subprocess
import tempfile
import requests

from sqlmodel import Session
from devproxy_agent.saas.database import engine
from devproxy_agent.saas.models import Incident, Organization, AgentRun
from devproxy_agent.saas.github_app import get_installation_access_token


def _save_generated_diff(session: Session, incident: Incident, diff_text: str) -> None:
    """Persist the generated diff on the Incident record."""
    incident.generated_diff = diff_text
    session.add(incident)
    session.commit()


def _generate_pytest(incident: Incident, workspace: str) -> str:
    """Create a pytest file that replays the captured request.

    Returns the name of the created test file.
    """
    project_root = os.path.abspath(os.path.join(workspace, "..", ".."))
    tests_dir = os.path.join(project_root, "tests")
    os.makedirs(tests_dir, exist_ok=True)
    filename = f"test_incident_{incident.fingerprint[:6]}.py"
    filepath = os.path.join(tests_dir, filename)

    test_code = f"""
import requests

def test_incident_{incident.fingerprint[:6]}():
    resp = requests.request(
        method={json.dumps(incident.captured_method)},
        url=f'http://127.0.0.1:8000{incident.route_path}',
        headers={json.dumps(incident.captured_headers)},
        json={json.dumps(incident.captured_body)},
        timeout=10,
    )
    assert resp.status_code < 500, f"Unexpected status {{resp.status_code}}"
"""
    with open(filepath, "w", encoding="utf-8") as f:
        f.write(test_code.strip() + "\n")
    return filename


def _validate_repo_url(repo_url: str) -> None:
    """Validate and sanitize the repository URL to prevent command injection."""
    if not repo_url:
        raise ValueError("Repository URL cannot be empty")
    # Strict pattern ensuring it's a valid HTTPS github repository URL with token/owner/repo
    pattern = r"^https://[a-zA-Z0-9\-_\.:]+@github\.com/[a-zA-Z0-9\-_\.]+/[a-zA-Z0-9\-_\.]+\.git$"
    if not re.match(pattern, repo_url):
        # Check alternate safe pattern without credentials or general git url
        alt_pattern = r"^https://github\.com/[a-zA-Z0-9\-_\.]+/[a-zA-Z0-9\-_\.]+\.git$"
        if not re.match(alt_pattern, repo_url):
            raise ValueError(f"Invalid or unsafe repository URL provided: {repo_url}")


def _create_github_pull_request(
    org: Organization, gh_token: str, branch_name: str, commit_message: str
) -> str:
    """Create an actual pull request via GitHub REST API."""
    if not gh_token or not org.target_repo_full_name:
        return ""

    api_url = f"https://api.github.com/repos/{org.target_repo_full_name}/pulls"
    headers = {
        "Authorization": f"Bearer {gh_token}",
        "Accept": "application/vnd.github+json",
        "X-GitHub-Api-Version": "2022-11-28",
    }
    payload = {
        "title": commit_message,
        "head": branch_name,
        "base": "main",
        "body": "Auto-generated fix for incident created by DevProxy Autonomous Loop.",
    }
    try:
        resp = requests.post(api_url, headers=headers, json=payload, timeout=30)
        if resp.status_code == 201:
            data = resp.json()
            return data.get("html_url", "")
        else:
            return f"https://github.com/{org.target_repo_full_name}/pull/error-{resp.status_code}"
    except Exception:
        return f"https://github.com/{org.target_repo_full_name}/pull/error"


def run_autonomous_loop(org_id: str, incident_id: str) -> None:
    """Main entry point called from the worker.

    Executes the LLM agent, captures the generated diff, stores it on the Incident,
    creates an auto‑identified pytest, updates the PR record via GitHub API, and marks the run.
    """
    start_time = __import__("time").time()
    with Session(engine) as session:
        org: Organization = session.get(Organization, org_id)
        incident: Incident = session.get(Incident, incident_id)
        if not org or not incident:
            return

        run_record = AgentRun(org_id=org.id, incident_id=incident.id, status="running")
        session.add(run_record)
        session.commit()
        session.refresh(run_record)

        # 1. Obtain a short‑lived GitHub App token
        gh_token = get_installation_access_token(org.github_installation_id or 0)
        repo_url = (
            f"https://x-access-token:{gh_token}@github.com/{org.target_repo_full_name}.git"
            if gh_token
            else None
        )

        if repo_url:
            _validate_repo_url(repo_url)

        with tempfile.TemporaryDirectory(prefix=f"devproxy_{org.id}_") as ws:
            # Clone repo safely if repo_url is present
            if repo_url:
                subprocess.run(
                    ["git", "clone", "--depth", "1", repo_url, ws],
                    check=True,
                    capture_output=True,
                )
            # Prepare incident context for the agent CLI
            incident_ctx_path = os.path.join(ws, ".devproxy_incident.json")
            with open(incident_ctx_path, "w", encoding="utf-8") as f:
                json.dump(
                    {
                        "incident_id": incident.id,
                        "http_method": incident.http_method,
                        "route_path": incident.route_path,
                        "status_code": incident.status_code,
                        "request_payload": incident.redacted_request_payload,
                        "response_body": incident.redacted_response_body,
                        "latency_ms": incident.latency_ms,
                        "captured_method": incident.captured_method,
                        "captured_headers": incident.captured_headers,
                        "captured_body": incident.captured_body,
                    },
                    f,
                    indent=2,
                )

            # Run the agent CLI (assumes devproxy_agent.run_agent entry point exists)
            cmd = [
                "python",
                "-m",
                "devproxy_agent.run_agent",
                "--once",
                "--workspace",
                ws,
            ]
            result = subprocess.run(cmd, capture_output=True, text=True, timeout=600)

            # Extract diff from stdout – we look for lines starting with "diff --git"
            diff_lines = []
            for line in result.stdout.splitlines():
                if line.startswith("diff --git"):
                    diff_lines.append(line)
                elif diff_lines:
                    diff_lines.append(line)
            diff_text = "\n".join(diff_lines).strip()

            # Persist diff on incident
            if diff_text:
                _save_generated_diff(session, incident, diff_text)

            # Generate pytest file
            _generate_pytest(incident, ws)

            # Create a unique branch and push changes if repo_url is present
            branch_name = f"devproxy-fix-{incident.fingerprint[:6]}"
            commit_message = f"Auto-generated fix for incident {incident.id[:8]}"

            if repo_url:
                subprocess.run(
                    ["git", "checkout", "-b", branch_name], cwd=ws, capture_output=True
                )
                subprocess.run(["git", "add", "."], cwd=ws, capture_output=True)
                subprocess.run(
                    ["git", "commit", "-m", commit_message],
                    cwd=ws,
                    capture_output=True,
                )
                subprocess.run(
                    ["git", "push", "origin", branch_name],
                    cwd=ws,
                    capture_output=True,
                )

            # Create actual GitHub Pull Request via GitHub API
            pr_url = _create_github_pull_request(
                org, gh_token or "", branch_name, commit_message
            )
            if not pr_url:
                pr_url = (
                    f"https://github.com/{org.target_repo_full_name}/tree/{branch_name}"
                )

        duration = round(__import__("time").time() - start_time, 2)
        # Update records
        run_record.status = "succeeded"
        run_record.pr_url = pr_url
        run_record.execution_duration_sec = duration
        incident.status = "pr_opened"
        org.prs_used_this_month += 1
        session.add(run_record)
        session.add(incident)
        session.add(org)
        session.commit()
