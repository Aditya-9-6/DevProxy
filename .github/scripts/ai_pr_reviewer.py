#!/usr/bin/env python3
"""
Autonomous PR Validator & Reviewer Agent for DevProxy
Inspects Pull Request diffs against main, conducts security, concurrency,
performance, and test coverage audits using Google Gemini AI, and posts
a structured review directly to the PR.
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

DEFAULT_MODEL = "gemini-3.1-flash-lite"
FALLBACK_MODELS = [
    "gemini-3.1-flash-lite",
    "gemini-3.5-flash",
    "gemini-3.8-flash",
    "gemini-3.7-flash",
]

REVIEWER_SYSTEM_PROMPT = """You are an elite principal systems software architect and security auditor reviewing Pull Requests on DevProxy (a high-performance HTTP/HTTPS proxy and security engine in Go).

Your task is to provide an objective, thorough, actionable code review of the pull request.

CRITICAL AUDIT INVARIANTS:
1. Concurrency Safety: Data races, goroutine leaks, improper mutex unlocking (defer), context cancellation propagation.
2. Memory & Performance: Zero-allocation pooling (sync.Pool), streaming buffers, avoidance of large in-memory byte slice copies on hot proxy paths.
3. Security & Input Sanitization: SSRF, request smuggling, path traversal, command injection, TLS certificate verification bypasses.
4. Go Idioms & Quality: Proper error wrapping, sentinel errors, table-driven tests, race detection.

OUTPUT FORMAT:
Generate your response in clean, professional GitHub-flavored Markdown. Include:
## 🔍 Autonomous Architecture & Security Review
- **Overall Verdict**: [✅ APPROVED | ⚠️ ACTION REQUIRED | 💬 INFORMATIONAL]
- **Executive Summary**: 2-3 sentences summarizing the PR impact.

### 🛡️ Security & Concurrency Audit
- Analysis of data race freedom, goroutine lifecycles, and security boundaries.

### ⚡ Performance & Streaming Invariants
- Memory allocation footprint, streaming compliance, zero-copy buffer handling.

### 🧪 Test Coverage & Edge Cases
- Coverage assessment, missing boundary/negative test cases.

### 💡 Suggestions & Recommendations
- Specific, actionable code suggestions if improvements are needed.
"""

def call_gemini(api_key: str, prompt: str, fallback_key: str = "", model: str = DEFAULT_MODEL) -> str:
    """Calls Gemini REST API with fallback models and fallback API key."""
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
                }
            }

            print(f"[*] Requesting PR review from model: {current_model}...", flush=True)
            max_attempts = 3
            for attempt in range(1, max_attempts + 1):
                req = urllib.request.Request(
                    url,
                    data=json.dumps(payload).encode("utf-8"),
                    headers={"Content-Type": "application/json"},
                    method="POST"
                )
                try:
                    with urllib.request.urlopen(req, timeout=90) as resp:
                        data = json.loads(resp.read().decode("utf-8"))
                        text_response = data["candidates"][0]["content"]["parts"][0]["text"].strip()
                        return text_response
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
            return res.stdout.strip()[:30000]
    except Exception:
        pass
    try:
        res = subprocess.run(["git", "diff", "HEAD~1"], cwd=workspace, capture_output=True, text=True)
        if res.stdout.strip():
            return res.stdout.strip()[:30000]
    except Exception:
        pass
    return "No git diff available."

def main():
    parser = argparse.ArgumentParser(description="DevProxy Autonomous PR Reviewer")
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

    prompt = f"""Review the following Pull Request for DevProxy:

### Pull Request Title:
{title}

### Pull Request Description:
{body}

### Git Diff (origin/main...HEAD):
```diff
{diff}
```

Conduct a comprehensive review following the specified criteria.
"""

    review_md = call_gemini(primary_key, prompt, fallback_key=fallback_key)

    review_file = workspace / "ai_pr_review.md"
    footer = f"\n\n---\n*Automated review generated by DevProxy AI Reviewer Bot. Comment `/review` to trigger a re-audit.*"
    review_file.write_text(review_md + footer, encoding="utf-8")
    print(f"[OK] Review generated successfully in {review_file}")

if __name__ == "__main__":
    main()
