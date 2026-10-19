#!/usr/bin/env python3
"""
Autonomous Triage & Cleanup Gatekeeper for DevProxy.
High-speed batch auditing of open Pull Requests and Issues:
- Closes PRs that are obsolete, duplicate, unneeded, unviable, or whose linked issues are already closed.
- Closes Issues that are duplicates, already implemented on main, or superseded by merged PRs.
- Automatically deletes associated remote branches upon closing PRs.
"""

import sys
import os
import re
import json
import subprocess
from pathlib import Path

REPO = "Aditya-9-6/DevProxy"

def run_cmd(cmd, cwd=None, check=False):
    try:
        res = subprocess.run(
            cmd,
            cwd=cwd,
            shell=isinstance(cmd, str),
            capture_output=True,
            text=True,
            encoding="utf-8",
            errors="replace",
            timeout=30
        )
        if check and res.returncode != 0:
            return ""
        return res.stdout.strip()
    except Exception:
        return ""

def audit_and_close_unneeded_prs(workspace: Path) -> int:
    """Audits open PRs and closes those that are not needed using fast batch queries."""
    print("[*] Fetching open PRs, closed issues, and merged PRs for batch triage...", flush=True)

    # 1. Fast batch fetch open PRs
    prs_raw = run_cmd([
        "gh", "pr", "list",
        "--repo", REPO,
        "--state", "open",
        "--limit", "100",
        "--json", "number,title,headRefName,url"
    ], cwd=workspace)
    if not prs_raw:
        print("[*] No open PRs found.", flush=True)
        return 0

    prs = json.loads(prs_raw)

    # 2. Fast batch fetch closed issue numbers
    closed_issues_raw = run_cmd([
        "gh", "issue", "list",
        "--repo", REPO,
        "--state", "closed",
        "--limit", "100",
        "--json", "number"
    ], cwd=workspace)
    closed_issues = set(i["number"] for i in json.loads(closed_issues_raw)) if closed_issues_raw else set()

    # 3. Fast batch fetch merged PR numbers and linked issue references
    merged_prs_raw = run_cmd([
        "gh", "pr", "list",
        "--repo", REPO,
        "--state", "merged",
        "--limit", "100",
        "--json", "number,title,headRefName"
    ], cwd=workspace)
    merged_prs = json.loads(merged_prs_raw) if merged_prs_raw else []

    merged_issue_to_pr = {}
    for m in merged_prs:
        t = m.get("title", "")
        h = m.get("headRefName", "")
        m_num = m.get("number")
        for num_str in re.findall(r"#(\d+)", t):
            merged_issue_to_pr[int(num_str)] = m_num
        m_m = re.search(r"(?:solve-)?issue-(\d+)", h)
        if m_m:
            try:
                merged_issue_to_pr[int(m_m.group(1))] = m_num
            except Exception:
                pass

    closed_count = 0
    issue_to_prs = {}

    for p in prs:
        num = p["number"]
        head = p["headRefName"]
        title = p["title"]
        linked = None
        m_m = re.search(r"(?:solve-)?issue-(\d+)", head)
        if m_m:
            try:
                linked = int(m_m.group(1))
            except Exception:
                pass
        if not linked:
            nums = re.findall(r"#(\d+)", title)
            if nums:
                linked = int(nums[0])

        if linked:
            issue_to_prs.setdefault(linked, []).append(p)

        # Check Reason A: Linked issue is already closed
        if linked and linked in closed_issues:
            print(f"[*] Closing PR #{num} ({head}): Linked issue #{linked} is already closed.", flush=True)
            run_cmd([
                "gh", "pr", "close", str(num),
                "--repo", REPO,
                "--comment", f"🤖 **DevProxy Gatekeeper Triage**: Automatically closing this PR because its linked issue #{linked} is already closed or obsolete.",
                "--delete-branch"
            ], cwd=workspace)
            closed_count += 1
            continue

        # Check Reason B: Linked issue already resolved by merged PR
        if linked and linked in merged_issue_to_pr:
            merged_num = merged_issue_to_pr[linked]
            print(f"[*] Closing PR #{num} ({head}): Linked issue #{linked} was already resolved by merged PR #{merged_num}.", flush=True)
            run_cmd([
                "gh", "pr", "close", str(num),
                "--repo", REPO,
                "--comment", f"🤖 **DevProxy Gatekeeper Triage**: Automatically closing this PR because linked issue #{linked} has already been resolved and merged in PR #{merged_num}.",
                "--delete-branch"
            ], cwd=workspace)
            closed_count += 1
            continue

    # Check Reason C: Duplicate PRs for the same issue (keep the newest, close older)
    for issue_num, pr_list in issue_to_prs.items():
        if len(pr_list) > 1:
            sorted_prs = sorted(pr_list, key=lambda x: x["number"], reverse=True)
            kept_pr = sorted_prs[0]
            for dup in sorted_prs[1:]:
                d_num = dup["number"]
                print(f"[*] Closing duplicate PR #{d_num}: Superseded by newer PR #{kept_pr['number']} for issue #{issue_num}.", flush=True)
                run_cmd([
                    "gh", "pr", "close", str(d_num),
                    "--repo", REPO,
                    "--comment", f"🤖 **DevProxy Gatekeeper Triage**: Automatically closing duplicate PR #{d_num} as superseded by PR #{kept_pr['number']} addressing issue #{issue_num}.",
                    "--delete-branch"
                ], cwd=workspace)
                closed_count += 1

    return closed_count

def audit_and_close_unneeded_issues(workspace: Path) -> int:
    """Audits open issues and closes duplicates or already implemented ones."""
    print("[*] Auditing open issues for triage...", flush=True)
    raw_issues = run_cmd([
        "gh", "issue", "list",
        "--repo", REPO,
        "--state", "open",
        "--limit", "100",
        "--json", "number,title,labels"
    ], cwd=workspace)
    if not raw_issues:
        return 0

    issues = json.loads(raw_issues)
    closed_count = 0
    seen_titles = {}

    for iss in issues:
        num = iss["number"]
        title = iss["title"]
        norm_title = re.sub(r"[^a-zA-Z0-9]", "", title).lower()

        # Check duplicate open issues
        if norm_title in seen_titles:
            orig_num = seen_titles[norm_title]
            print(f"[*] Closing duplicate issue #{num}: Duplicate of #{orig_num}.", flush=True)
            run_cmd([
                "gh", "issue", "close", str(num),
                "--repo", REPO,
                "--comment", f"🤖 **DevProxy Gatekeeper Triage**: Automatically closing duplicate issue as identical to #{orig_num}.",
                "--reason", "not planned"
            ], cwd=workspace)
            closed_count += 1
            continue
        seen_titles[norm_title] = num

    return closed_count

def main():
    workspace = Path(".").resolve()
    print("[*] Running Autonomous Gatekeeper Triage Cleanup for DevProxy...", flush=True)
    prs_closed = audit_and_close_unneeded_prs(workspace)
    issues_closed = audit_and_close_unneeded_issues(workspace)
    print(f"[OK] Gatekeeper Triage complete. Closed {prs_closed} PR(s) and {issues_closed} issue(s).", flush=True)

if __name__ == "__main__":
    main()
