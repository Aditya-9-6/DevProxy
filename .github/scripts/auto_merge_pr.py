#!/usr/bin/env python3
"""
Autonomous PR Auto-Merger for DevProxy & GossipMesh.
Evaluates PR mergeability and CI check statuses:
If and only if ALL checks pass (100% green) and zero merge conflicts exist,
it auto-merges the PR to main and cleans up the branch.
"""

import sys
import os
import json
import time
import subprocess
import argparse
from pathlib import Path

def run_cmd(cmd: list[str], cwd: Path = Path(".")) -> tuple[int, str, str]:
    """Runs a subprocess command and returns (code, stdout, stderr)."""
    res = subprocess.run(cmd, cwd=cwd, capture_output=True, text=True, encoding="utf-8", errors="replace")
    return res.returncode, res.stdout.strip(), res.stderr.strip()

def get_pr_details(pr_number: str, workspace: Path) -> dict:
    """Fetches PR status, state, draft status, and mergeability with retries for UNKNOWN."""
    for attempt in range(1, 5):
        code, stdout, stderr = run_cmd([
            "gh", "pr", "view", pr_number,
            "--json", "number,title,state,isDraft,mergeable,headRefName,baseRefName"
        ], cwd=workspace)
        if code != 0:
            print(f"[!] Failed to view PR #{pr_number}: {stderr}", file=sys.stderr)
            return {}
        try:
            data = json.loads(stdout)
            if data.get("mergeable") != "UNKNOWN" or attempt == 4:
                return data
            print(f"[*] PR #{pr_number} mergeability is UNKNOWN. Waiting for GitHub calculation (attempt {attempt}/4)...", flush=True)
            time.sleep(4)
        except Exception as e:
            print(f"[!] JSON parse error: {e}", file=sys.stderr)
            return {}
    return {}

def evaluate_checks(pr_number: str, workspace: Path) -> tuple[list[str], list[str], list[str]]:
    """
    Evaluates PR status checks using `gh pr checks`.
    Returns (passing, failing, pending).
    """
    code, stdout, stderr = run_cmd([
        "gh", "pr", "checks", pr_number,
        "--json", "name,bucket,state"
    ], cwd=workspace)
    if code != 0 or not stdout:
        print(f"[*] No check runs recorded yet or checks query failed: {stderr}", flush=True)
        return [], [], []

    try:
        checks = json.loads(stdout)
    except Exception as e:
        print(f"[!] Failed to parse checks JSON: {e}", file=sys.stderr)
        return [], [], []

    passing = []
    failing = []
    pending = []

    for c in checks:
        name = c.get("name", "Unknown Check")
        bucket = (c.get("bucket") or "").lower()
        state = (c.get("state") or "").upper()

        # Exclude auto-merge and reviewer workflows from blocking themselves
        if any(ignored in name.lower() for ignored in ["auto-merge", "auto merge"]):
            continue

        if bucket == "fail" or state in ("FAILURE", "CANCELLED", "TIMED_OUT", "ACTION_REQUIRED"):
            failing.append(f"{name} ({state or bucket})")
        elif bucket == "pending" or state in ("PENDING", "STARTING", "QUEUED", "IN_PROGRESS"):
            pending.append(f"{name} ({state or bucket})")
        elif bucket in ("pass", "skipping") or state in ("SUCCESS", "SKIPPED", "NEUTRAL"):
            passing.append(name)
        else:
            pending.append(f"{name} ({state or bucket})")

    return passing, failing, pending

def auto_merge_pr(pr_number: str, workspace: Path, notify_on_failure: bool = False) -> bool:
    """Evaluates PR conditions and merges if 100% green and no conflicts."""
    print(f"[*] Evaluating Auto-Merge Gate for PR #{pr_number} in {workspace}...", flush=True)
    pr = get_pr_details(pr_number, workspace)
    if not pr:
        print(f"[!] Could not retrieve metadata for PR #{pr_number}.", file=sys.stderr)
        return False

    title = pr.get("title", "")
    state = pr.get("state", "").upper()
    is_draft = pr.get("isDraft", False)
    mergeable = (pr.get("mergeable") or "UNKNOWN").upper()
    head_ref = pr.get("headRefName", "")
    base_ref = pr.get("baseRefName", "main")

    if state != "OPEN":
        print(f"[*] PR #{pr_number} is not OPEN (current state: {state}). Skipping merge.", flush=True)
        return False

    if is_draft:
        print(f"[*] PR #{pr_number} is currently a Draft. Skipping merge.", flush=True)
        return False

    passing, failing, pending = evaluate_checks(pr_number, workspace)
    print(f"[*] PR #{pr_number} Check Results: {len(passing)} Passing, {len(failing)} Failing, {len(pending)} Pending.", flush=True)
    print(f"[*] PR #{pr_number} Mergeability: {mergeable}", flush=True)

    has_conflicts = (mergeable == "CONFLICTING")
    is_mergeable = (mergeable == "MERGEABLE")
    can_merge = is_mergeable and len(failing) == 0 and len(pending) == 0 and len(passing) > 0

    if can_merge:
        print(f"\n[🎉 READY TO MERGE] PR #{pr_number} meets all conditions! Merging to '{base_ref}'...", flush=True)
        # Attempt squash merge first, fallback to regular merge
        code, stdout, stderr = run_cmd([
            "gh", "pr", "merge", pr_number,
            "--squash", "--delete-branch",
            "--subject", f"{title} (#{pr_number})"
        ], cwd=workspace)

        if code != 0:
            print(f"[*] Squash merge failed ({stderr}). Attempting standard merge commit...", flush=True)
            code, stdout, stderr = run_cmd([
                "gh", "pr", "merge", pr_number,
                "--merge", "--delete-branch"
            ], cwd=workspace)

        if code == 0:
            print(f"[SUCCESS] Successfully auto-merged PR #{pr_number} into {base_ref}!", flush=True)
            success_comment = f"""🎉 **Pull Request #{pr_number} Auto-Merged!**

- **Mergeability**: Clean (0 conflicts)
- **CI Validation**: {len(passing)} check(s) passed (100% GREEN)
- **Branch**: `{head_ref}` merged into `{base_ref}` and deleted.
"""
            run_cmd(["gh", "pr", "comment", pr_number, "--body", success_comment], cwd=workspace)
            return True
        else:
            print(f"[!] Merge execution failed: {stderr}", file=sys.stderr)
            if notify_on_failure:
                err_comment = f"⚠️ **Auto-Merge Failed**: GitHub rejected merge command:\n```text\n{stderr}\n```"
                run_cmd(["gh", "pr", "comment", pr_number, "--body", err_comment], cwd=workspace)
            return False

    # PR cannot be merged at this time
    print(f"[!] PR #{pr_number} cannot be merged yet.", flush=True)
    if has_conflicts:
        print(f"    - Blocked: Merge conflicts detected ({mergeable}).", flush=True)
    if failing:
        print(f"    - Blocked: {len(failing)} failing check(s): {failing}", flush=True)
    if pending:
        print(f"    - Blocked: {len(pending)} pending/running check(s): {pending}", flush=True)
    if len(passing) == 0:
        print("    - Blocked: Zero passing checks recorded.", flush=True)

    if notify_on_failure:
        fail_list = "\n".join(f"  - ❌ `{f}`" for f in failing) if failing else "  - None"
        pend_list = "\n".join(f"  - ⏳ `{p}`" for p in pending) if pending else "  - None"
        comment = f"""⚠️ **Auto-Merge Pre-conditions Not Met for PR #{pr_number}**

- **Merge Status**: `{mergeable}` {"(⚠️ Conflicts detected!)" if has_conflicts else "(Clean)"}
- **Passing Checks**: {len(passing)}

**Failing Checks:**
{fail_list}

**Pending / In-Progress Checks:**
{pend_list}

> 💡 *A PR will only auto-merge when 100% of CI checks pass and zero merge conflicts exist.*
> Comment `/fix` to trigger the autonomous repair and conflict resolution agent.
"""
        run_cmd(["gh", "pr", "comment", pr_number, "--body", comment], cwd=workspace)

    return False

def main():
    parser = argparse.ArgumentParser(description="Autonomous PR Auto-Merger")
    parser.add_argument("--pr-number", required=True, help="GitHub Pull Request Number")
    parser.add_argument("--workspace", default=".", help="Workspace path")
    parser.add_argument("--notify-on-failure", action="store_true", help="Post comment explaining blockers if merge fails")
    args = parser.parse_args()

    workspace = Path(args.workspace).resolve()
    merged = auto_merge_pr(args.pr_number, workspace, notify_on_failure=args.notify_on_failure)
    sys.exit(0 if merged else 1)

if __name__ == "__main__":
    main()
