#!/usr/bin/env python3
"""
Autonomous Multi-Agent GitHub Advancement Loop for DevProxy (Local Daemon & Cloud Runner)

Executes the continuous hyper-scale development cycle:
1. Dynamic Key Pooling: Rotates Gemini API keys round-robin to maximize RPM/TPM quota.
2. Issue Generation: Creates architecture advancement specifications (strictly deduplicated).
3. AI Solver: Implements solutions with zero-allocation, anti-spaghetti invariants.
4. Pre-Commit Quality Gate: Runs gofmt -w ., go vet ./..., and go test -race ./...
5. Contribution Attribution: Commits with 'Aditya Dahale <aditya-9-6@users.noreply.github.com>'.
6. Autonomous Reviewer Gate: Audits PR against strict performance and decoupling invariants.
7. Auto-Merger: Squash-merges approved PRs to continuously illuminate GitHub contribution chart,
   and explicitly closes linked issues to prevent duplicate solutions.
"""

import os
import sys
import json
import re
import time
import subprocess
import argparse
from pathlib import Path

try:
    sys.stdout.reconfigure(line_buffering=True, encoding="utf-8", errors="replace")
    sys.stderr.reconfigure(line_buffering=True, encoding="utf-8", errors="replace")
except Exception:
    pass

REPO = "Aditya-9-6/DevProxy"
USER_NAME = "Aditya Dahale"
USER_EMAIL = "aditya-9-6@users.noreply.github.com"

# Base Key Pool - Dynamically loaded from GEMINI_KEY_POOL, individual env vars, or .gemini_keys
DEFAULT_KEYS = []

class KeyPool:
    """Manages round-robin rotation and dynamic failover across Gemini API keys."""
    def __init__(self, initial_keys: list[str] = None):
        self.keys = []
        pool_env = os.environ.get("GEMINI_KEY_POOL", "")
        extra_keys = [k.strip() for k in pool_env.split(",") if k.strip()]

        for var in ("GEMINI_API_KEY", "GEMINI_ISSUE_KEY", "GEMINI_REVIEWER_KEY", "GEMINI_SOLVER_KEY"):
            val = os.environ.get(var, "").strip()
            if val and val not in extra_keys:
                extra_keys.append(val)

        # Check local gitignored .gemini_keys file in workspace root
        for check_dir in (Path("."), Path(__file__).resolve().parent, Path(__file__).resolve().parent.parent):
            key_file = check_dir / ".gemini_keys"
            if key_file.exists():
                try:
                    for line in key_file.read_text(encoding="utf-8").splitlines():
                        k = line.strip()
                        if k and not k.startswith("#") and k not in extra_keys:
                            extra_keys.append(k)
                except Exception:
                    pass

        all_keys = extra_keys + (initial_keys or [])
        for k in all_keys:
            if k and k not in self.keys:
                self.keys.append(k)
        self.idx = 0

    def add_keys(self, new_keys: list[str]):
        for k in new_keys:
            clean = k.strip()
            if clean and clean not in self.keys:
                self.keys.append(clean)
                print(f"[KeyPool] Added new API key to active rotation pool (Total: {len(self.keys)})", flush=True)

    def next_key(self) -> str:
        if not self.keys:
            return ""
        key = self.keys[self.idx % len(self.keys)]
        self.idx += 1
        return key

    def __len__(self):
        return len(self.keys)

GLOBAL_POOL = KeyPool(DEFAULT_KEYS)
FAILED_SOLVE_ATTEMPTS: dict[int, int] = {}

def run_cmd(cmd, cwd=None, check=True):
    """Runs a shell command and returns stdout."""
    res = subprocess.run(cmd, cwd=cwd, shell=isinstance(cmd, str), capture_output=True, text=True, encoding="utf-8", errors="replace")
    if check and res.returncode != 0:
        raise RuntimeError(f"Command failed ({res.returncode}): {cmd}\nStderr: {res.stderr}\nStdout: {res.stdout}")
    return res.stdout.strip()

def get_open_advancement_issues(workspace: Path) -> list:
    """
    Fetches open advancement issues, strictly filtering out any that
    already have active PRs or were already solved by previous merged PRs.
    """
    try:
        out = run_cmd(["gh", "issue", "list", "--repo", REPO, "--label", "advancement", "--state", "open", "--json", "number,title,body"], cwd=workspace)
        all_issues = json.loads(out) if out else []
    except Exception as e:
        print(f"[Warning] Failed to fetch issues: {e}", file=sys.stderr, flush=True)
        return []

    if not all_issues:
        return []

    # Fetch recent PRs to check for active or merged solutions
    try:
        prs_out = run_cmd(["gh", "pr", "list", "--repo", REPO, "--state", "all", "--limit", "50", "--json", "number,title,headRefName,state"], cwd=workspace)
        prs = json.loads(prs_out) if prs_out else []
    except Exception:
        prs = []

    active_issue_nums = set()
    merged_issue_nums = set()

    for p in prs:
        head = p.get("headRefName", "")
        title = p.get("title", "")
        p_state = p.get("state", "").upper()

        cand_num = None
        if "issue-" in head:
            try:
                cand_num = int(head.split("issue-")[-1].split("-")[0])
            except Exception:
                pass
        if not cand_num:
            match = re.search(r"#(\d+)", title)
            if match:
                cand_num = int(match.group(1))

        if cand_num:
            if p_state == "OPEN":
                active_issue_nums.add(cand_num)
            elif p_state == "MERGED":
                merged_issue_nums.add(cand_num)

    unaddressed = []
    for iss in all_issues:
        num = iss.get("number")
        if num in merged_issue_nums:
            print(f"[*] Issue #{num} is already solved by a merged PR. Closing issue on GitHub...", flush=True)
            run_cmd(["gh", "issue", "close", str(num), "--repo", REPO, "--comment", "Closed autonomously as already solved by a merged PR."], cwd=workspace, check=False)
            continue
        if num in active_issue_nums:
            print(f"[*] Issue #{num} already has an active open PR in progress. Skipping duplicate solve.", flush=True)
            continue
        unaddressed.append(iss)

    return unaddressed

def generate_new_issue(workspace: Path):
    """Runs the issue generator script with rotated key."""
    active_key = GLOBAL_POOL.next_key()
    print(f"[*] Generating new architecture advancement issue for DevProxy (Key index: {GLOBAL_POOL.idx % len(GLOBAL_POOL)})...", flush=True)
    env = os.environ.copy()
    env["GEMINI_ISSUE_KEY"] = active_key
    env["GH_REPO"] = REPO
    res = subprocess.run(
        [sys.executable, ".github/scripts/generate_advancement_issue.py", "--force"],
        cwd=workspace,
        env=env,
        capture_output=True,
        text=True,
        encoding="utf-8",
        errors="replace"
    )
    print(res.stdout, flush=True)
    if res.returncode != 0:
        print(f"[!] Generator error: {res.stderr}", file=sys.stderr, flush=True)

def run_diagnostics(workspace: Path) -> tuple[int, str]:
    """Runs Go compiler diagnostics, vet, and unit tests."""
    out_lines = []
    subprocess.run(["go", "mod", "tidy"], cwd=workspace, capture_output=True, text=True)

    fmt_res = subprocess.run(["gofmt", "-l", "."], cwd=workspace, capture_output=True, text=True)
    if fmt_res.stdout.strip():
        out_lines.append("=== GOFMT FORMATTING ERRORS ===")
        out_lines.append(fmt_res.stdout.strip())

    vet_res = subprocess.run(["go", "vet", "./..."], cwd=workspace, capture_output=True, text=True)
    if vet_res.returncode != 0:
        out_lines.append("=== GO VET COMPILATION ERRORS ===")
        out_lines.append(vet_res.stderr.strip() or vet_res.stdout.strip())

    test_res = subprocess.run(["go", "test", "-v", "./..."], cwd=workspace, capture_output=True, text=True)
    if test_res.returncode != 0:
        out_lines.append("=== GO TEST FAILURES ===")
        out_lines.append((test_res.stdout + "\n" + test_res.stderr).strip())

    combined = "\n".join(out_lines).strip()
    return (0 if not combined else 1), combined

def heal_and_merge_open_prs(workspace: Path) -> bool:
    """Finds all open AI PRs, self-heals failing checks or /fix comments, reviews, and merges them."""
    try:
        out = run_cmd([
            "gh", "pr", "list",
            "--repo", REPO,
            "--state", "open",
            "--json", "number,title,headRefName,url,comments"
        ], cwd=workspace)
        prs = json.loads(out) if out else []
    except Exception as e:
        print(f"[Warning] Failed to fetch open PRs: {e}", file=sys.stderr)
        return False

    if not prs:
        return False

    def is_candidate_pr(p):
        ref = p.get("headRefName", "")
        title = p.get("title", "")
        comments = [c.get("body", "") for c in p.get("comments", [])]
        has_fix = any("/fix" in c for c in comments)
        return (
            ref.startswith("ai/")
            or "feat(ai)" in title
            or has_fix
        )

    # Prioritize AI feature branches first, then dependabot/external
    ai_prs = sorted(
        [p for p in prs if is_candidate_pr(p)],
        key=lambda p: 0 if p.get("headRefName", "").startswith("ai/") else 1
    )
    if not ai_prs:
        return False

    print(f"\n[PR Sweeper] Found {len(ai_prs)} open candidate PR(s). Checking for self-healing and auto-merge...", flush=True)
    try:
        for pr in ai_prs:
            pr_num = pr["number"]
            head_ref = pr["headRefName"]
            title = pr["title"]
            print(f"\n[*] Evaluating PR #{pr_num}: {title} (branch: {head_ref})...", flush=True)

            # Determine linked issue number
            linked_issue = None
            if "issue-" in head_ref:
                try:
                    linked_issue = int(head_ref.split("issue-")[-1].split("-")[0])
                except Exception:
                    pass
            if not linked_issue:
                match = re.search(r"#(\d+)", title)
                if match:
                    linked_issue = int(match.group(1))

            # If the linked issue is already closed in GitHub, close this PR as obsolete!
            if linked_issue:
                try:
                    iss_out = run_cmd(["gh", "issue", "view", str(linked_issue), "--repo", REPO, "--json", "state"], cwd=workspace, check=False)
                    if iss_out and '"state":"CLOSED"' in iss_out:
                        print(f"[*] PR #{pr_num} resolves Issue #{linked_issue} which is ALREADY CLOSED. Closing obsolete PR...", flush=True)
                        run_cmd(["gh", "pr", "close", str(pr_num), "--repo", REPO, "--comment", f"Closed obsolete PR: Issue #{linked_issue} is already resolved and closed.", "--delete-branch"], cwd=workspace, check=False)
                        continue
                except Exception:
                    pass

            try:
                run_cmd("git reset --hard HEAD", cwd=workspace, check=False)
                run_cmd("git clean -fd", cwd=workspace, check=False)
                run_cmd(["git", "fetch", "origin", head_ref], cwd=workspace, check=False)
                run_cmd(["git", "checkout", "-B", head_ref, f"origin/{head_ref}"], cwd=workspace)
                merge_res = subprocess.run(["git", "merge", "origin/main", "--no-edit"], cwd=workspace, capture_output=True, text=True)
                if merge_res.returncode != 0:
                    run_cmd(["git", "checkout", "origin/main", "--", ".github/", "autonomous_loop.py"], cwd=workspace, check=False)
                    run_cmd("git add -A", cwd=workspace, check=False)
                    run_cmd(["git", "commit", "-m", "merge: resolve tooling conflicts with origin/main", "--no-edit"], cwd=workspace, check=False)
            except Exception as e:
                print(f"[!] Could not checkout branch {head_ref}: {e}", file=sys.stderr)
                continue

            code, error_log = run_diagnostics(workspace)
            has_fix_request = any("/fix" in c.get("body", "") for c in pr.get("comments", []))

            if code != 0 or has_fix_request:
                print(f"[*] PR #{pr_num} requires repair (exit code: {code}, /fix requested: {has_fix_request}). Launching autonomous fixer...", flush=True)
                fix_env = os.environ.copy()
                fix_env["GEMINI_SOLVER_KEY"] = GLOBAL_POOL.next_key()
                fix_env["GH_REPO"] = REPO

                subprocess.run([
                    sys.executable, ".github/scripts/ai_ci_fixer.py",
                    "--pr-number", str(pr_num),
                    "--workspace", str(workspace),
                    "--max-iterations", "3"
                ], cwd=workspace, env=fix_env)

                code, error_log = run_diagnostics(workspace)
                if code == 0:
                    print(f"[SUCCESS] PR #{pr_num} repaired to 100% GREEN! Committing and pushing...", flush=True)
                    run_cmd(["git", "config", "--replace-all", "user.name", USER_NAME], cwd=workspace, check=False)
                    run_cmd(["git", "config", "--replace-all", "user.email", USER_EMAIL], cwd=workspace, check=False)
                    run_cmd("git rm --cached -f ai_pr_*.md ai_review_*.json ci_fix_*.md 2>/dev/null || true", cwd=workspace, check=False)
                    run_cmd("git add -A", cwd=workspace)
                    run_cmd("git reset -- ai_pr_*.md ai_review_*.json ci_fix_*.md 2>/dev/null || true", cwd=workspace, check=False)
                    run_cmd([
                        "git", "commit",
                        "-m", f"fix(ci): autonomous 100% green self-healing repair for PR #{pr_num}",
                        "-m", f"Co-authored-by: {USER_NAME} <{USER_EMAIL}>"
                    ], cwd=workspace, check=False)
                    run_cmd(["git", "push", "origin", head_ref], cwd=workspace, check=False)
                    run_cmd(["gh", "pr", "comment", str(pr_num), "--repo", REPO, "--body", f"✅ **Autonomous Fix Applied!** Diagnostics passed 100% GREEN on branch `{head_ref}`."], cwd=workspace, check=False)

            if code == 0:
                print(f"[*] Running Reviewer on PR #{pr_num}...", flush=True)
                rev_env = os.environ.copy()
                rev_env["GEMINI_REVIEWER_KEY"] = GLOBAL_POOL.next_key()
                rev_env["GH_REPO"] = REPO
                subprocess.run([
                    sys.executable, ".github/scripts/ai_pr_reviewer.py",
                    "--pr-number", str(pr_num),
                    "--workspace", str(workspace)
                ], cwd=workspace, env=rev_env)

                status_file = workspace / "ai_review_status.json"
                score = 0
                verdict = "ACTION_REQUIRED"
                if status_file.exists():
                    try:
                        s_data = json.loads(status_file.read_text(encoding="utf-8"))
                        score = s_data.get("score", 0)
                        verdict = s_data.get("verdict", "ACTION_REQUIRED")
                    except Exception:
                        pass

                if verdict != "APPROVED" or score < 90:
                    print(f"[*] PR #{pr_num} review scored {score}/100. Applying reviewer-guided self-healing...", flush=True)
                    fix_env = os.environ.copy()
                    fix_env["GEMINI_SOLVER_KEY"] = GLOBAL_POOL.next_key()
                    fix_env["GH_REPO"] = REPO

                    subprocess.run([
                        sys.executable, ".github/scripts/ai_ci_fixer.py",
                        "--pr-number", str(pr_num),
                        "--workspace", str(workspace),
                        "--max-iterations", "3"
                    ], cwd=workspace, env=fix_env)

                    c2, _ = run_diagnostics(workspace)
                    if c2 == 0:
                        run_cmd(["git", "config", "--replace-all", "user.name", USER_NAME], cwd=workspace, check=False)
                        run_cmd(["git", "config", "--replace-all", "user.email", USER_EMAIL], cwd=workspace, check=False)
                        run_cmd("git rm --cached -f ai_pr_*.md ai_review_*.json ci_fix_*.md 2>/dev/null || true", cwd=workspace, check=False)
                        run_cmd("git add -A", cwd=workspace)
                        run_cmd("git reset -- ai_pr_*.md ai_review_*.json ci_fix_*.md 2>/dev/null || true", cwd=workspace, check=False)
                        run_cmd([
                            "git", "commit",
                            "-m", f"fix(review): apply architectural improvements for PR #{pr_num}",
                            "-m", f"Co-authored-by: {USER_NAME} <{USER_EMAIL}>"
                        ], cwd=workspace, check=False)
                        run_cmd(["git", "push", "origin", head_ref], cwd=workspace, check=False)

                        # Re-run reviewer to verify approval
                        rev_env["GEMINI_REVIEWER_KEY"] = GLOBAL_POOL.next_key()
                        subprocess.run([
                            sys.executable, ".github/scripts/ai_pr_reviewer.py",
                            "--pr-number", str(pr_num),
                            "--workspace", str(workspace)
                        ], cwd=workspace, env=rev_env)

                        if status_file.exists():
                            try:
                                s_data = json.loads(status_file.read_text(encoding="utf-8"))
                                score = s_data.get("score", 0)
                                verdict = s_data.get("verdict", "ACTION_REQUIRED")
                            except Exception:
                                pass

                if verdict == "APPROVED" and score >= 90:
                    print(f"[APPROVED] PR #{pr_num} APPROVED ({score}/100)! Merging autonomously...", flush=True)
                    run_cmd(["gh", "label", "create", "ready-to-merge", "--repo", REPO, "--color", "0E8A16", "-f"], cwd=workspace, check=False)
                    run_cmd(["gh", "pr", "edit", str(pr_num), "--repo", REPO, "--add-label", "ready-to-merge"], cwd=workspace, check=False)
                    res = subprocess.run(["gh", "pr", "merge", str(pr_num), "--repo", REPO, "--squash", "--admin"], cwd=workspace, capture_output=True, text=True)
                    if res.returncode != 0:
                        subprocess.run(["gh", "pr", "merge", str(pr_num), "--repo", REPO, "--squash"], cwd=workspace)
                    print(f"[MERGED] Merged PR #{pr_num} into main! Contributor activity recorded for {USER_NAME}.", flush=True)

                    # Explicitly close linked issue to prevent duplicate solve
                    if linked_issue:
                        print(f"[*] Closing linked Issue #{linked_issue}...", flush=True)
                        run_cmd(["gh", "issue", "close", str(linked_issue), "--repo", REPO, "--comment", f"Resolved and closed autonomously after merging PR #{pr_num}."], cwd=workspace, check=False)

                    run_cmd("git checkout main", cwd=workspace)
                    run_cmd("git pull origin main", cwd=workspace)
                    run_cmd(f"git branch -D {head_ref}", cwd=workspace, check=False)
                    run_cmd(f"git push origin --delete {head_ref}", cwd=workspace, check=False)
                    return True
    finally:
        run_cmd("git checkout main", cwd=workspace, check=False)
    return False

def solve_issue(workspace: Path, issue_num: int, issue_title: str, issue_body: str):
    """Solves an issue, creates PR, reviews, and merges."""
    solver_key = GLOBAL_POOL.next_key()
    print(f"\n=======================================================", flush=True)
    print(f"[*] Solving Issue #{issue_num}: {issue_title}", flush=True)
    print(f"[*] Active Solver API Key: ...{solver_key[-6:] if len(solver_key) > 6 else 'key'}", flush=True)
    print(f"=======================================================", flush=True)

    env = os.environ.copy()
    env["GEMINI_API_KEY"] = solver_key
    env["GH_REPO"] = REPO

    solver_cmd = [
        sys.executable, ".github/scripts/ai_issue_solver.py",
        "--issue-number", str(issue_num),
        "--issue-title", issue_title,
        "--issue-body", issue_body,
        "--workspace", str(workspace)
    ]
    res = subprocess.run(solver_cmd, cwd=workspace, env=env, capture_output=True, text=True, encoding="utf-8", errors="replace")
    print(res.stdout, flush=True)
    if res.returncode != 0:
        print(f"[!] Solver failed: {res.stderr}", file=sys.stderr, flush=True)
        return False

    # Pre-commit Quality Assurance Gate
    print("[*] Tidying Go modules with go mod tidy...", flush=True)
    run_cmd(["go", "mod", "tidy"], cwd=workspace, check=False)

    print("[*] Formatting Go codebase with gofmt...", flush=True)
    run_cmd(["gofmt", "-w", "."], cwd=workspace, check=False)

    print("[*] Verifying Go compilation & static analysis (go vet)...", flush=True)
    vet_res = subprocess.run(["go", "vet", "./..."], cwd=workspace, capture_output=True, text=True, encoding="utf-8", errors="replace")

    print("[*] Verifying Go concurrency & race safety (go test -race)...", flush=True)
    check_res = subprocess.run(["go", "test", "-race", "./..."], cwd=workspace, capture_output=True, text=True, encoding="utf-8", errors="replace")

    if vet_res.returncode != 0 or check_res.returncode != 0:
        print("[*] Pre-commit gate detected build/test issues. Triggering autonomous compiler repair pass...", flush=True)
        fix_env = os.environ.copy()
        fix_env["GEMINI_SOLVER_KEY"] = solver_key
        fix_env["GEMINI_API_KEY"] = solver_key
        fixer_cmd = [
            sys.executable, ".github/scripts/ai_ci_fixer.py",
            "--workspace", str(workspace),
            "--pr-number", str(issue_num)
        ]
        fix_proc = subprocess.run(fixer_cmd, cwd=workspace, env=fix_env, capture_output=True, text=True, encoding="utf-8", errors="replace")
        print(fix_proc.stdout, flush=True)
        run_cmd(["go", "mod", "tidy"], cwd=workspace, check=False)
        run_cmd(["gofmt", "-w", "."], cwd=workspace, check=False)
        vet_res = subprocess.run(["go", "vet", "./..."], cwd=workspace, capture_output=True, text=True, encoding="utf-8", errors="replace")
        check_res = subprocess.run(["go", "test", "-race", "./..."], cwd=workspace, capture_output=True, text=True, encoding="utf-8", errors="replace")

    if vet_res.returncode != 0:
        print(f"[!] go vet failed: {vet_res.stderr}", file=sys.stderr, flush=True)
        run_cmd("git reset --hard HEAD", cwd=workspace, check=False)
        run_cmd("git clean -fd", cwd=workspace, check=False)
        return False

    if check_res.returncode != 0:
        print(f"[!] go test failed: {check_res.stderr}", file=sys.stderr, flush=True)
        run_cmd("git reset --hard HEAD", cwd=workspace, check=False)
        run_cmd("git clean -fd", cwd=workspace, check=False)
        return False

    branch_name = f"ai/solve-issue-{issue_num}"
    print(f"[*] Staging changes on branch {branch_name}...", flush=True)
    run_cmd(f"git checkout -B {branch_name}", cwd=workspace)
    run_cmd(["git", "config", "--replace-all", "user.name", USER_NAME], cwd=workspace, check=False)
    run_cmd(["git", "config", "--replace-all", "user.email", USER_EMAIL], cwd=workspace, check=False)

    run_cmd("git rm --cached -f ai_pr_*.md ai_review_*.json ci_fix_*.md 2>/dev/null || true", cwd=workspace, check=False)
    run_cmd("git add -A", cwd=workspace)
    run_cmd("git reset -- ai_pr_*.md ai_review_*.json ci_fix_*.md 2>/dev/null || true", cwd=workspace, check=False)

    commit_msg = f"feat: automated resolution for issue #{issue_num} ({issue_title})\n\nCloses #{issue_num}"
    run_cmd([
        "git", "commit",
        "-m", commit_msg,
        "-m", f"Co-authored-by: {USER_NAME} <{USER_EMAIL}>"
    ], cwd=workspace, check=False)

    print(f"[*] Pushing branch {branch_name} to origin...", flush=True)
    run_cmd(f"git push origin {branch_name} --force", cwd=workspace)

    print("[*] Creating / updating Pull Request...", flush=True)
    summary_file = workspace / "ai_pr_summary.md"
    body_content = summary_file.read_text(encoding="utf-8") if summary_file.exists() else f"Automated resolution for Issue #{issue_num}."
    if f"Closes #{issue_num}" not in body_content and f"Fixes #{issue_num}" not in body_content:
        body_content += f"\n\n---\n*Closes #{issue_num}*"

    pr_url = ""
    pr_num = None
    try:
        existing = run_cmd(["gh", "pr", "list", "--repo", REPO, "--head", branch_name, "--json", "number,url"], cwd=workspace)
        prs = json.loads(existing) if existing else []
        if prs:
            pr_num = prs[0]["number"]
            pr_url = prs[0]["url"]
            print(f"[OK] Found existing PR #{pr_num}: {pr_url}", flush=True)
        else:
            pr_out = run_cmd([
                "gh", "pr", "create",
                "--repo", REPO,
                "--head", branch_name,
                "--base", "main",
                "--title", f"feat(ai): resolve #{issue_num} - {issue_title}",
                "--body", body_content,
                "--label", "advancement,ai-generated"
            ], cwd=workspace)
            pr_url = pr_out.strip()
            pr_num = int(pr_url.split("/")[-1])
            print(f"[OK] Created PR #{pr_num}: {pr_url}", flush=True)
    except Exception as e:
        print(f"[!] Failed to manage PR: {e}", file=sys.stderr, flush=True)
        return False

    reviewer_key = GLOBAL_POOL.next_key()
    print(f"[*] Running Autonomous Architectural Reviewer on PR #{pr_num} (Key index: {GLOBAL_POOL.idx % len(GLOBAL_POOL)})...", flush=True)
    env["GEMINI_REVIEWER_KEY"] = reviewer_key
    rev_res = subprocess.run([
        sys.executable, ".github/scripts/ai_pr_reviewer.py",
        "--pr-number", str(pr_num),
        "--workspace", str(workspace)
    ], cwd=workspace, env=env, capture_output=True, text=True, encoding="utf-8", errors="replace")
    print(rev_res.stdout, flush=True)

    review_status_file = workspace / "ai_review_status.json"
    score = 0
    verdict = "ACTION_REQUIRED"
    if review_status_file.exists():
        try:
            status_data = json.loads(review_status_file.read_text(encoding="utf-8"))
            score = status_data.get("score", 0)
            verdict = status_data.get("verdict", "ACTION_REQUIRED")
        except Exception:
            pass

    review_md_file = workspace / "ai_pr_review.md"
    if review_md_file.exists():
        run_cmd(["gh", "pr", "comment", str(pr_num), "--repo", REPO, "--body-file", str(review_md_file)], cwd=workspace, check=False)

    if verdict == "APPROVED" and score >= 90:
        print(f"[APPROVED] PR #{pr_num} APPROVED (Score: {score}/100)! Merging into main...", flush=True)
        run_cmd(["gh", "label", "create", "ready-to-merge", "--repo", REPO, "--color", "0E8A16", "-f"], cwd=workspace, check=False)
        run_cmd(["gh", "pr", "edit", str(pr_num), "--repo", REPO, "--add-label", "ready-to-merge"], cwd=workspace, check=False)
        res = subprocess.run(["gh", "pr", "merge", str(pr_num), "--repo", REPO, "--squash", "--admin"], cwd=workspace, capture_output=True, text=True)
        if res.returncode != 0:
            subprocess.run(["gh", "pr", "merge", str(pr_num), "--repo", REPO, "--squash"], cwd=workspace)
        print(f"[SUCCESS] Merged PR #{pr_num} into main! Contributor activity recorded for {USER_NAME}.", flush=True)

        # Explicitly close linked issue on GitHub
        print(f"[*] Closing linked Issue #{issue_num}...", flush=True)
        run_cmd(["gh", "issue", "close", str(issue_num), "--repo", REPO, "--comment", f"Resolved and closed autonomously after merging PR #{pr_num}."], cwd=workspace, check=False)

        run_cmd("git checkout main", cwd=workspace)
        run_cmd("git pull origin main", cwd=workspace)
        # Clean up feature branch
        run_cmd(f"git branch -D {branch_name}", cwd=workspace, check=False)
        run_cmd(f"git push origin --delete {branch_name}", cwd=workspace, check=False)
        return True
    else:
        print(f"[!] PR #{pr_num} verdict: {verdict} (Score: {score}). Awaiting review fixes.", flush=True)
        return False

def run_loop_iteration(workspace: Path):
    """Executes a single cycle of the autonomous loop."""
    print(f"\n--- [Autonomous DevProxy Loop Iteration: {time.strftime('%Y-%m-%d %H:%M:%S')} | Key Pool: {len(GLOBAL_POOL)} keys] ---", flush=True)

    # Synchronize with GossipMesh
    try:
        from gossipmesh import GossipNode, MemeticKnowledgeBase
        g_node = GossipNode("devproxy_daemon")
        new_gossips = g_node.sync()
        g_node.heartbeat()
        kb = MemeticKnowledgeBase()
        top_memes = kb.get_top_memes(repo="DevProxy")
        print(f"[*] GossipMesh: Synced {len(top_memes)} living architectural memes ({len(new_gossips)} incoming peer digests).", flush=True)
    except Exception:
        pass

    # Phase 1: Heal and merge existing open PRs
    handled_pr = heal_and_merge_open_prs(workspace)
    if handled_pr:
        print("[OK] Processed open PRs in this cycle.", flush=True)

    # Phase 2: Replenish and solve open advancement issues
    issues = get_open_advancement_issues(workspace)

    # Maintain continuous backlog of at least 3 active advancement issues
    TARGET_BACKLOG = 3
    if len(issues) < TARGET_BACKLOG:
        needed = TARGET_BACKLOG - len(issues)
        print(f"[*] Advancement issue backlog low ({len(issues)}/{TARGET_BACKLOG}). Generating {needed} new issue(s)...", flush=True)
        for _ in range(needed):
            generate_new_issue(workspace)
            time.sleep(3)
        issues = get_open_advancement_issues(workspace)

    if not issues:
        print("[!] No issues available to solve.", flush=True)
        return

    # Select candidate issue that has not repeatedly failed
    target = None
    for iss in issues:
        num = iss.get("number")
        if FAILED_SOLVE_ATTEMPTS.get(num, 0) < 2:
            target = iss
            break

    if not target:
        print("[*] All currently open issues have exceeded failure threshold. Generating fresh issue...", flush=True)
        generate_new_issue(workspace)
        time.sleep(3)
        issues = get_open_advancement_issues(workspace)
        for iss in issues:
            if FAILED_SOLVE_ATTEMPTS.get(iss.get("number"), 0) < 2:
                target = iss
                break

    if not target:
        target = issues[0]

    num = target["number"]
    print(f"[*] Selected Issue #{num} for autonomous resolution: '{target['title']}'", flush=True)
    success = solve_issue(workspace, target["number"], target["title"], target.get("body", ""))
    if success:
        FAILED_SOLVE_ATTEMPTS.pop(num, None)
    else:
        FAILED_SOLVE_ATTEMPTS[num] = FAILED_SOLVE_ATTEMPTS.get(num, 0) + 1
        print(f"[!] Issue #{num} failed solve attempt ({FAILED_SOLVE_ATTEMPTS[num]}/2).", flush=True)

def main():
    parser = argparse.ArgumentParser(description="DevProxy Autonomous Multi-Agent Daemon")
    parser.add_argument("--workspace", default=".", help="Path to DevProxy repository root")
    parser.add_argument("--once", action="store_true", help="Run once and exit instead of continuous daemon")
    parser.add_argument("--interval", type=int, default=10, help="Interval in seconds between cycles (default: 10s)")
    parser.add_argument("--max-cycles", type=int, default=0, help="Maximum number of cycles to execute (0 = unlimited)")
    parser.add_argument("--timeout-mins", type=int, default=0, help="Maximum minutes to run before exiting (0 = unlimited)")
    parser.add_argument("--add-keys", nargs="*", default=[], help="Additional Gemini API keys to add to the round-robin pool")
    args = parser.parse_args()

    workspace = Path(args.workspace).resolve()

    if args.add_keys:
        GLOBAL_POOL.add_keys(args.add_keys)

    print(f"[*] DevProxy Multi-Agent Engine initialized with {len(GLOBAL_POOL)} API keys in active rotation.", flush=True)

    if args.once:
        run_loop_iteration(workspace)
    else:
        print(f"[*] Starting DevProxy Continuous Autonomous Daemon (interval: {args.interval}s, max_cycles: {args.max_cycles or 'unlimited'}, timeout: {args.timeout_mins or 'unlimited'}m)...", flush=True)
        start_time = time.time()
        cycles = 0
        while True:
            try:
                run_loop_iteration(workspace)
                cycles += 1
                if args.max_cycles and cycles >= args.max_cycles:
                    print(f"[OK] Completed target max cycles ({cycles}). Exiting batch gracefully.", flush=True)
                    break
                if args.timeout_mins and (time.time() - start_time) >= args.timeout_mins * 60:
                    print(f"[OK] Reached batch time limit ({args.timeout_mins}m). Exiting batch gracefully.", flush=True)
                    break
            except Exception as e:
                print(f"[Error in loop]: {e}", file=sys.stderr, flush=True)
            print(f"[*] Sleeping for {args.interval} seconds until next iteration...", flush=True)
            time.sleep(args.interval)

if __name__ == "__main__":
    main()
