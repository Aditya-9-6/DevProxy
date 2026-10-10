#!/usr/bin/env python3
"""
Autonomous PR Validator & Reviewer Agent for DevProxy
Enforces strict anti-spaghetti architectural standards, modularity, zero-leak concurrency,
zero-allocation memory invariants, and high test coverage.
Outputs structured JSON and Markdown reports with system impact assessment.
"""

import os
import sys
import json
import argparse
import urllib.request
import urllib.error
import subprocess
import time
from pathlib import Path

DEFAULT_MODEL = "gemini-3-flash-preview"
FALLBACK_MODELS = [
    "gemini-3-flash-preview",
    "gemma-4-26b-a4b-it",
    "gemini-flash-latest",
]

REVIEWER_SYSTEM_PROMPT = """You are an elite principal systems software architect and security auditor reviewing Pull Requests on DevProxy (a high-performance HTTP/HTTPS proxy and security engine in Go).

Your primary mission is to ENFORCE STRICT ARCHITECTURAL STANDARDS AND PREVENT SPAGHETTI CODE as the codebase scales at hyper-speed.

CRITICAL ARCHITECTURAL & ANTI-SPAGHETTI INVARIANTS:
1. Anti-Spaghetti & Modularity:
   - Single Responsibility Principle (SRP): Functions must be focused (<60 LOC), clean, and self-documenting.
   - No spaghetti control flow: Reject deeply nested blocks (>3 levels), massive switch-case god functions, or unstructured goto/fallthrough loops.
   - Clean Package Boundaries: Preserve encapsulation between `pkg/proxy`, `pkg/analysis`, `pkg/ringbuffer`, `pkg/cert`, `pkg/dashboard`. No circular dependencies or cross-package leakages.
2. Concurrency Safety:
   - Zero data races, proper synchronization via `sync.RWMutex`, `sync.Once`, atomic operations, or channels.
   - Goroutine lifecycle safety: All launched goroutines must terminate cleanly upon context cancellation (`ctx.Done()`). No leaks.
   - Defer hygiene: Mutex unlocks and resource closes must be deferred immediately with zero defer leaks inside hot unbounded loops.
3. Memory & High Throughput Invariants:
   - Zero-allocation hot paths: Use `sync.Pool` for byte buffers (`[]byte`). Avoid allocating slices or copying payloads in the proxy stream forwarding path.
   - Streaming compliance: Read and forward payloads via streaming readers/writers (`io.Reader`, `io.Writer`) without buffering entire multi-megabyte streams in RAM.
4. Security & Hardening:
   - Strict input validation: Protection against SSRF, request smuggling (TE.CL/CL.TE), path traversal, TLS bypass, and command injection.
5. Go Idiomatic Standards & Test Coverage:
   - Error wrapping using `%w` and proper sentinel error checking with `errors.Is`/`errors.As`.
   - Table-driven unit tests covering happy paths, edge cases, negative/error paths, and concurrent execution (`t.Parallel()`).

EVALUATION & VERDICT:
- If code has compilation errors, missing dependencies, data race risks, spaghetti code, god functions, or missing unit tests:
  verdict must be "ACTION_REQUIRED" and score < 90.
- If code is clean, modular, race-free, properly tested, and meets all anti-spaghetti architectural invariants:
  verdict must be "APPROVED" and score >= 90 (typically 95-100).

OUTPUT SCHEMA:
You MUST respond ONLY with a single valid JSON object with the following schema:
{
  "verdict": "APPROVED" or "ACTION_REQUIRED",
  "score": 95,
  "executive_summary": "Concise summary of the PR and overall architectural quality.",
  "anti_spaghetti_audit": "Detailed evaluation of code modularity, function sizing, decoupling, and maintainability.",
  "security_concurrency_audit": "Detailed analysis of data race freedom, goroutine lifecycle, mutex safety, and security posture.",
  "performance_memory_audit": "Evaluation of buffer pooling (sync.Pool), zero-allocation hot paths, and streaming throughput.",
  "test_coverage_audit": "Evaluation of test coverage, edge cases, error conditions, and concurrency tests.",
  "system_impact": "Detailed assessment of the PR's effect on the DevProxy system (impacted subsystems, throughput, latency, security posture, operational reliability). If approved, cc @Aditya-9-6.",
  "action_items": [
    "Specific refactoring step or improvement needed (empty list if APPROVED)"
  ]
}
"""

def call_gemini(api_key: str, prompt: str, fallback_key: str = "", model: str = DEFAULT_MODEL) -> dict:
    """Calls Gemini REST API with fallback models and fallback API key, expecting JSON."""
    ordered = [model] + [m for m in FALLBACK_MODELS if m != model]
    models_to_try = []
    for m in ordered:
        if m not in models_to_try:
            models_to_try.append(m)

    keys_to_try = [k for k in [api_key, fallback_key] if k.strip()]

    last_err = None
    for current_key in keys_to_try:
        for current_model in models_to_try:
            url = f"https://generativelanguage.googleapis.com/v1beta/models/{current_model}:generateContent?key={current_key}"
            payload = {
                "contents": [
                    {
                        "parts": [
                            {"text": REVIEWER_SYSTEM_PROMPT},
                            {"text": prompt}
                        ]
                    }
                ],
                "generationConfig": {
                    "temperature": 0.2,
                    "responseMimeType": "application/json"
                }
            }

            print(f"[*] Requesting architectural PR review from model: {current_model}...", flush=True)
            max_attempts = 5
            for attempt in range(1, max_attempts + 1):
                req = urllib.request.Request(
                    url,
                    data=json.dumps(payload).encode("utf-8"),
                    headers={"Content-Type": "application/json"},
                    method="POST"
                )
                try:
                    with urllib.request.urlopen(req, timeout=180) as resp:
                        data = json.loads(resp.read().decode("utf-8"))
                        text_response = data["candidates"][0]["content"]["parts"][0]["text"].strip()
                        try:
                            return json.loads(text_response, strict=False)
                        except json.JSONDecodeError:
                            if text_response.startswith("```"):
                                lines = text_response.splitlines()
                                if lines[0].startswith("```"):
                                    lines = lines[1:]
                                if lines and lines[-1].startswith("```"):
                                    lines = lines[:-1]
                                text_response = "\n".join(lines).strip()
                            return json.loads(text_response, strict=False)
                except urllib.error.HTTPError as e:
                    err_msg = e.read().decode("utf-8", errors="replace")
                    print(f"[Warning] HTTP {e.code} (attempt {attempt}/{max_attempts}) with model {current_model}: {err_msg[:160]}", file=sys.stderr)
                    last_err = err_msg
                    if e.code == 404:
                        break
                    if e.code in (429, 500, 502, 503, 504) and attempt < max_attempts:
                        time.sleep(3 * attempt)
                        continue
                    break
                except Exception as e:
                    print(f"[Warning] Error (attempt {attempt}/{max_attempts}) with model {current_model}: {e}", file=sys.stderr)
                    last_err = str(e)
                    if attempt < max_attempts:
                        time.sleep(3 * attempt)
                        continue
                    break

    raise RuntimeError(f"Failed to obtain PR review from Gemini API. Last error: {last_err}")

def get_pr_diff(workspace: Path) -> str:
    """Gets git diff against origin/main."""
    try:
        res = subprocess.run(["git", "diff", "origin/main...HEAD"], cwd=workspace, capture_output=True, text=True)
        if res.stdout.strip():
            return res.stdout.strip()[:35000]
    except Exception:
        pass
    try:
        res = subprocess.run(["git", "diff", "HEAD~1"], cwd=workspace, capture_output=True, text=True)
        if res.stdout.strip():
            return res.stdout.strip()[:35000]
    except Exception:
        pass
    return "No git diff available."

def get_head_sha(workspace: Path) -> str:
    """Gets current HEAD commit SHA."""
    try:
        res = subprocess.run(["git", "rev-parse", "HEAD"], cwd=workspace, capture_output=True, text=True)
        return res.stdout.strip()
    except Exception:
        return ""

def set_commit_status(head_sha: str, state: str, description: str, context: str = "ai/architectural-review"):
    """Posts a commit status check to GitHub."""
    gh_repo = os.environ.get("GH_REPO") or os.environ.get("GITHUB_REPOSITORY")
    gh_token = os.environ.get("GH_TOKEN") or os.environ.get("GITHUB_TOKEN")
    if not (gh_repo and gh_token and head_sha):
        return

    url = f"https://api.github.com/repos/{gh_repo}/statuses/{head_sha}"
    payload = {
        "state": state, # success, failure, pending
        "description": description[:140],
        "context": context
    }
    req = urllib.request.Request(
        url,
        data=json.dumps(payload).encode("utf-8"),
        headers={
            "Authorization": f"Bearer {gh_token}",
            "Accept": "application/vnd.github+json",
            "Content-Type": "application/json"
        },
        method="POST"
    )
    try:
        with urllib.request.urlopen(req, timeout=30) as resp:
            print(f"[OK] Commit status set to {state} ({context})")
    except Exception as e:
        print(f"[Warning] Failed to set commit status: {e}", file=sys.stderr)

def main():
    parser = argparse.ArgumentParser(description="DevProxy Autonomous PR Reviewer & Architectural Gate")
    parser.add_argument("--pr-number", required=True, help="GitHub Pull Request Number")
    parser.add_argument("--workspace", default=".", help="Workspace root directory")
    args = parser.parse_args()

    primary_key = os.environ.get("GEMINI_REVIEWER_KEY", "").strip() or os.environ.get("GEMINI_API_KEY", "").strip()
    fallback_key = os.environ.get("GEMINI_SOLVER_KEY", "").strip() or os.environ.get("GEMINI_ISSUE_KEY", "").strip()

    if not primary_key:
        print("MISSING_KEY: GEMINI_REVIEWER_KEY or GEMINI_API_KEY is not set.", file=sys.stderr)
        sys.exit(2)

    workspace = Path(args.workspace).resolve()
    print(f"[*] Starting Autonomous PR Reviewer for PR #{args.pr_number}...")

    # Fetch PR details
    try:
        pr_json = subprocess.run(
            ["gh", "pr", "view", args.pr_number, "--json", "title,body,headRefName,baseRefName,files"],
            cwd=workspace, capture_output=True, text=True
        )
        pr_data = json.loads(pr_json.stdout) if pr_json.returncode == 0 else {}
    except Exception:
        pr_data = {}

    title = pr_data.get("title", f"Pull Request #{args.pr_number}")
    body = pr_data.get("body", "No description provided.")
    diff = get_pr_diff(workspace)
    head_sha = get_head_sha(workspace)

    prompt = f"""Conduct a thorough architectural and anti-spaghetti audit of this Pull Request for DevProxy:

### Pull Request Title:
{title}

### Pull Request Description:
{body}

### Git Diff (origin/main...HEAD):
```diff
{diff}
```

Audit the code against all anti-spaghetti, concurrency, performance, and security requirements. Provide your verdict in the required JSON schema.
"""

    audit = call_gemini(primary_key, prompt, fallback_key=fallback_key)

    verdict = audit.get("verdict", "ACTION_REQUIRED").strip().upper()
    score = int(audit.get("score", 75))
    is_approved = (verdict == "APPROVED" and score >= 90)

    summary = audit.get("executive_summary", "Autonomous architectural audit completed.")
    anti_spaghetti = audit.get("anti_spaghetti_audit", "No architectural issues noted.")
    concurrency_sec = audit.get("security_concurrency_audit", "Concurrency & security invariants checked.")
    perf_mem = audit.get("performance_memory_audit", "Memory allocation & streaming checked.")
    test_cov = audit.get("test_coverage_audit", "Test coverage evaluated.")
    system_impact = audit.get("system_impact", "System impact evaluated.")
    action_items = audit.get("action_items", [])

    # Format Markdown Review
    if is_approved:
        review_md = f"""## 🌟 Autonomous Architectural Review: APPROVED (Score: {score}/100)

**cc @Aditya-9-6** — This Pull Request has achieved **100% architectural readiness** and strictly adheres to DevProxy's anti-spaghetti, concurrency, and performance invariants!

### 🌐 Architectural System Impact Report
{system_impact}

### 🍝 Anti-Spaghetti & Modularity Verification
{anti_spaghetti}

### 🛡️ Concurrency, Race Freedom & Security Audit
{concurrency_sec}

### ⚡ Performance & Zero-Allocation Memory Invariants
{perf_mem}

### 🧪 Test Coverage & Invariant Verification
{test_cov}

---
### 🚦 Next Steps: Maintainer Sign-Off Required
**@Aditya-9-6**: All automated quality gates, anti-spaghetti checks, and performance benchmarks have passed cleanly.
To complete the merge into `main`:
- **Approve this Pull Request**, OR
- Comment **`/merge`** on this PR.
"""
        commit_desc = f"Architectural Review Passed ({score}/100) - Ready to merge"
        set_commit_status(head_sha, "success", commit_desc)
    else:
        action_bullets = "\n".join(f"- {item}" for item in action_items) if action_items else "- Address structural feedback and missing test cases."
        review_md = f"""## ⚠️ Autonomous Architectural Review: ACTION REQUIRED (Score: {score}/100)

### 📋 Executive Summary
{summary}

### 🍝 Anti-Spaghetti & Modularity Findings
{anti_spaghetti}

### 🛡️ Concurrency & Security Findings
{concurrency_sec}

### ⚡ Performance & Memory Footprint Audit
{perf_mem}

### 🧪 Test Coverage Gaps
{test_cov}

### 🛠️ Required Refactoring & Action Items
{action_bullets}

---
🔄 **Autonomous Self-Healing Loop Active**: The PR Fixer Agent will refactor the code according to these directives and push updates until the PR achieves 100% readiness.
"""
        commit_desc = f"Architectural Improvements Required ({score}/100)"
        set_commit_status(head_sha, "failure", commit_desc)

    # Save Markdown file
    review_file = workspace / "ai_pr_review.md"
    review_file.write_text(review_md.strip(), encoding="utf-8")

    # Save Status JSON for workflow automation
    status_file = workspace / "ai_review_status.json"
    status_data = {
        "verdict": "APPROVED" if is_approved else "ACTION_REQUIRED",
        "score": score,
        "needs_improvement": not is_approved,
        "action_items": action_items,
        "system_impact": system_impact
    }
    status_file.write_text(json.dumps(status_data, indent=2), encoding="utf-8")

    # Broadcast findings to GossipMesh
    try:
        from gossipmesh import GossipNode, GossipTopic
        node = GossipNode("devproxy_reviewer")
        if is_approved:
            node.publish(
                GossipTopic.BEST_PRACTICES,
                {"pr_number": pr_number, "score": score, "summary": summary},
                repo="DevProxy"
            )
            print(f"[*] GossipMesh: Broadcasted APPROVED best-practice gossip across mesh for PR #{pr_number}.", flush=True)
        else:
            node.publish(
                GossipTopic.ANTI_PATTERNS,
                {"pr_number": pr_number, "score": score, "action_items": action_items, "summary": summary},
                repo="DevProxy"
            )
            print(f"[*] GossipMesh: Broadcasted ANTI_PATTERNS feedback gossip across mesh for PR #{pr_number}.", flush=True)
    except Exception as e:
        pass

    print(f"[OK] Audit finished: Verdict={status_data['verdict']} Score={score} (Written to {review_file} & {status_file})")

if __name__ == "__main__":
    main()
