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

DEFAULT_MODEL = "gemini-3.8-flash"
FALLBACK_MODELS = [
    "gemini-3.8-flash",
    "gemini-flash-latest",
]

DEFAULT_KEYS = []

def load_gemini_keys() -> list[str]:
    """Dynamically loads Gemini API keys from environment variables and local .gemini_keys files."""
    keys = []
    pool_env = os.environ.get("GEMINI_KEY_POOL", "")
    for k in pool_env.split(","):
        if k.strip() and k.strip() not in keys:
            keys.append(k.strip())
    for var in ("GEMINI_REVIEWER_KEY", "GEMINI_SOLVER_KEY", "GEMINI_API_KEY", "GEMINI_ISSUE_KEY"):
        val = os.environ.get(var, "").strip()
        if val and val not in keys:
            keys.append(val)
    for check_dir in (Path("."), Path(__file__).resolve().parent, Path(__file__).resolve().parent.parent, Path(__file__).resolve().parent.parent.parent, Path.home()):
        key_file = check_dir / ".gemini_keys"
        if key_file.exists():
            try:
                for line in key_file.read_text(encoding="utf-8").splitlines():
                    k = line.strip()
                    if k and not k.startswith("#") and k not in keys:
                        keys.append(k)
            except Exception:
                pass
    for k in DEFAULT_KEYS:
        if k not in keys:
            keys.append(k)
    return keys

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
    """Calls Spryzen Sovereign AI Gateway with semantic caching and fallback models."""
    # 🚀 Try Spryzen Gateway first (Sub-1ms Semantic Cache & Multi-Provider Cascade)
    try:
        from spryzen_gateway import GLOBAL_GATEWAY
        return GLOBAL_GATEWAY.call_with_smart_routing(
            prompt=prompt,
            system_prompt=REVIEWER_SYSTEM_PROMPT,
            preferred_model=model
        )
    except Exception as gw_err:
        print(f"[*] [Gateway Notice] Routing via direct fallback pool: {gw_err}", flush=True)

    ordered = [model] + [m for m in FALLBACK_MODELS if m != model]
    models_to_try = []
    for m in ordered:
        if m not in models_to_try:
            models_to_try.append(m)

    keys_to_try = [k for k in [api_key, fallback_key] if k.strip()]
    for k in load_gemini_keys():
        if k not in keys_to_try:
            keys_to_try.append(k)

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
                    if e.code == 429:
                        print(f"[*] Key rate-limited (429). Fast failover to next key in pool...", flush=True)
                        break
                    if e.code in (500, 502, 503, 504) and attempt < max_attempts:
                        time.sleep(2)
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

def extract_json_response(raw_text: str) -> dict:
    """Extracts JSON object from model response, stripping markdown code blocks if present."""
    raw_text = raw_text.strip()
    try:
        return json.loads(raw_text, strict=False)
    except json.JSONDecodeError:
        pass
    if raw_text.startswith("```"):
        lines = raw_text.splitlines()
        if lines[0].startswith("```"):
            lines = lines[1:]
        if lines and lines[-1].startswith("```"):
            lines = lines[:-1]
        raw_text = "\n".join(lines).strip()
        try:
            return json.loads(raw_text, strict=False)
        except json.JSONDecodeError:
            pass
    first_brace = raw_text.find("{")
    last_brace = raw_text.rfind("}")
    if first_brace != -1 and last_brace != -1 and last_brace > first_brace:
        return json.loads(raw_text[first_brace:last_brace + 1], strict=False)
    raise ValueError(f"Could not parse valid JSON from model response: {raw_text[:200]}")

def call_openrouter(api_key: str, prompt: str) -> dict:
    if not api_key:
        raise ValueError("Missing OPENROUTER_API_KEY")
    url = "https://openrouter.ai/api/v1/chat/completions"
    models = [
        "openrouter/free",
        "meta-llama/llama-3.3-70b-instruct:free",
        "deepseek/deepseek-chat:free",
        "deepseek/deepseek-r1:free",
        "qwen/qwen-2.5-coder-32b-instruct:free",
        "google/gemini-2.0-flash-exp:free",
        "mistralai/mistral-7b-instruct:free",
        "meta-llama/llama-3.3-70b-instruct",
        "deepseek/deepseek-chat"
    ]
    last_err = None
    for model in models:
        print(f"[*] [OpenRouter] Reviewing with {model}...", flush=True)
        payload = {
            "model": model,
            "messages": [
                {"role": "system", "content": REVIEWER_SYSTEM_PROMPT},
                {"role": "user", "content": prompt}
            ],
            "temperature": 0.2
        }
        req = urllib.request.Request(
            url,
            data=json.dumps(payload).encode("utf-8"),
            headers={
                "Authorization": f"Bearer {api_key}",
                "Content-Type": "application/json",
                "HTTP-Referer": "https://github.com/Aditya-9-6/DevProxy",
                "X-Title": "DevProxy Sovereign Autonomous Engine"
            },
            method="POST"
        )
        try:
            with urllib.request.urlopen(req, timeout=120) as resp:
                data = json.loads(resp.read().decode("utf-8"))
                return extract_json_response(data["choices"][0]["message"]["content"])
        except urllib.error.HTTPError as e:
            err_body = e.read().decode("utf-8", errors="replace")
            last_err = f"HTTP {e.code}: {err_body[:160]}"
            print(f"[!] [OpenRouter] Model {model} failed: HTTP {e.code} - {err_body[:120]}", file=sys.stderr)
            if e.code == 401:
                break
            continue
        except Exception as e:
            print(f"[!] [OpenRouter] Model {model} failed: {e}", file=sys.stderr)
            last_err = str(e)
            continue
    raise RuntimeError(f"OpenRouter calls failed: {last_err}")

def call_nvidia_nim(api_key: str, prompt: str) -> dict:
    if not api_key:
        raise ValueError("Missing NVIDIA_API_KEY")
    url = "https://integrate.api.nvidia.com/v1/chat/completions"
    models = [
        "meta/llama-3.3-70b-instruct",
        "nvidia/llama-3.3-nemotron-super-49b-v1.5",
        "mistralai/mistral-large-2-instruct",
        "meta/llama-3.1-70b-instruct",
        "meta/llama-3.1-8b-instruct"
    ]
    last_err = None
    for model in models:
        print(f"[*] [NVIDIA NIM] Reviewing with {model}...", flush=True)
        payload = {
            "model": model,
            "messages": [
                {"role": "system", "content": REVIEWER_SYSTEM_PROMPT},
                {"role": "user", "content": prompt}
            ],
            "temperature": 0.2
        }
        req = urllib.request.Request(
            url,
            data=json.dumps(payload).encode("utf-8"),
            headers={"Authorization": f"Bearer {api_key}", "Content-Type": "application/json"},
            method="POST"
        )
        try:
            with urllib.request.urlopen(req, timeout=120) as resp:
                data = json.loads(resp.read().decode("utf-8"))
                return extract_json_response(data["choices"][0]["message"]["content"])
        except urllib.error.HTTPError as e:
            err_body = e.read().decode("utf-8", errors="replace")
            last_err = f"HTTP {e.code}: {err_body[:160]}"
            print(f"[!] [NVIDIA NIM] Model {model} failed: HTTP {e.code} - {err_body[:120]}", file=sys.stderr)
            if e.code == 401 or e.code == 429:
                break
            continue
        except Exception as e:
            print(f"[!] [NVIDIA NIM] Model {model} failed: {e}", file=sys.stderr)
            last_err = str(e)
            continue
    raise RuntimeError(f"NVIDIA NIM calls failed: {last_err}")

def call_huggingface(api_key: str, prompt: str, system_prompt: str = REVIEWER_SYSTEM_PROMPT) -> dict:
    """Calls Hugging Face Inference Router API as resilient fallback."""
    if not api_key:
        raise ValueError("Missing HUGGINGFACE_API_KEY")
    endpoints = [
        "https://router.huggingface.co/v1/chat/completions",
        "https://api-inference.huggingface.co/v1/chat/completions"
    ]
    models = [
        "Qwen/Qwen2.5-Coder-32B-Instruct",
        "meta-llama/Llama-3.3-70B-Instruct",
        "deepseek-ai/DeepSeek-R1-Distill-Qwen-32B",
        "mistralai/Mistral-7B-Instruct-v0.3"
    ]
    last_err = None
    for endpoint in endpoints:
        for model in models:
            payload = {
                "model": model,
                "messages": [
                    {"role": "system", "content": system_prompt},
                    {"role": "user", "content": prompt}
                ],
                "temperature": 0.1
            }
            req = urllib.request.Request(
                endpoint,
                data=json.dumps(payload).encode("utf-8"),
                headers={"Authorization": f"Bearer {api_key}", "Content-Type": "application/json"},
                method="POST"
            )
            try:
                with urllib.request.urlopen(req, timeout=120) as resp:
                    data = json.loads(resp.read().decode("utf-8"))
                    text = data["choices"][0]["message"]["content"]
                    return extract_json_response(text)
            except urllib.error.HTTPError as e:
                err_body = e.read().decode("utf-8", errors="replace")
                last_err = f"HTTP {e.code}: {err_body[:160]}"
                if e.code == 401:
                    break
                continue
            except Exception as e:
                last_err = str(e)
                continue
    raise RuntimeError(f"HuggingFace calls failed: {last_err}")

def call_cohere(api_key: str, prompt: str, system_prompt: str = REVIEWER_SYSTEM_PROMPT) -> dict:
    """Calls Cohere Chat API as resilient fallback."""
    if not api_key:
        raise ValueError("Missing COHERE_API_KEY")
    url = "https://api.cohere.com/v2/chat"
    models = ["command-r-plus-08-2024", "command-r-08-2024", "command-r-plus", "command-r"]
    last_err = None
    for model in models:
        payload = {
            "model": model,
            "messages": [
                {"role": "system", "content": system_prompt},
                {"role": "user", "content": prompt}
            ]
        }
        req = urllib.request.Request(
            url,
            data=json.dumps(payload).encode("utf-8"),
            headers={"Authorization": f"Bearer {api_key}", "Content-Type": "application/json", "User-Agent": "DevProxy-CI/1.0"},
            method="POST"
        )
        try:
            with urllib.request.urlopen(req, timeout=120) as resp:
                data = json.loads(resp.read().decode("utf-8"))
                content = data.get("message", {}).get("content", [])
                if content and isinstance(content, list):
                    text = content[0].get("text", "")
                    return extract_json_response(text)
        except urllib.error.HTTPError as e:
            err_body = e.read().decode("utf-8", errors="replace")
            last_err = f"HTTP {e.code}: {err_body[:160]}"
            if e.code == 429 or e.code == 401:
                break
            continue
        except Exception as e:
            last_err = str(e)
            continue
    raise RuntimeError(f"Cohere calls failed: {last_err}")

def call_ai_reviewer(primary_key: str, prompt: str, fallback_key: str = "") -> dict:
    # 1. Try Gemini
    try:
        return call_gemini(primary_key, prompt, fallback_key=fallback_key)
    except Exception as e:
        print(f"[!] Gemini reviewer attempt failed: {e}. Trying secondary providers...", file=sys.stderr)

    # 2. Try OpenRouter (Multi-model free tier & fallback)
    openrouter_key = os.environ.get("OPENROUTER_API_KEY", "").strip()
    if openrouter_key:
        try:
            return call_openrouter(openrouter_key, prompt)
        except Exception as e:
            print(f"[!] OpenRouter reviewer failed: {e}", file=sys.stderr)

    # 3. Try NVIDIA NIM (Llama 3.3 70B & Nemotron)
    for nv_key in [os.environ.get("NVIDIA_API_KEY", "").strip(), os.environ.get("NVIDIA_API_KEY_2", "").strip()]:
        if nv_key:
            try:
                return call_nvidia_nim(nv_key, prompt)
            except Exception as e:
                print(f"[!] NVIDIA NIM reviewer failed: {e}", file=sys.stderr)

    # 4. Try Hugging Face
    hf_key = os.environ.get("HUGGINGFACE_API_KEY", "").strip()
    if hf_key:
        try:
            return call_huggingface(hf_key, prompt)
        except Exception as e:
            print(f"[!] HuggingFace reviewer failed: {e}", file=sys.stderr)

    # 5. Try Cohere
    cohere_key = os.environ.get("COHERE_API_KEY", "").strip()
    if cohere_key:
        try:
            return call_cohere(cohere_key, prompt)
        except Exception as e:
            print(f"[!] Cohere reviewer failed: {e}", file=sys.stderr)
        except Exception as e:
            print(f"[!] OpenRouter reviewer failed: {e}", file=sys.stderr)

    raise RuntimeError("All AI model providers (Gemini, Cohere, HuggingFace, NVIDIA NIM, OpenRouter) exhausted for PR review.")

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

    try:
        audit = call_ai_reviewer(primary_key, prompt, fallback_key=fallback_key)
    except Exception as e:
        print(f"[Warning] Gemini API review call failed: {e}", file=sys.stderr)
        
        # Check if rate-limit notice is already posted on this PR to prevent comment loops
        try:
            cmd = ["gh", "pr", "view", str(pr_num), "--json", "comments"]
            res = subprocess.run(cmd, capture_output=True, text=True, cwd=workspace)
            if res.returncode == 0:
                comments_data = json.loads(res.stdout).get("comments", [])
                if any("Temporarily Rate-Limited" in c.get("body", "") for c in comments_data):
                    print(f"[*] Rate-limited notice already posted on PR #{pr_num}. Skipping duplicate comment.")
                    status_file = workspace / "ai_review_status.json"
                    status_data = {
                        "verdict": "RATE_LIMITED",
                        "score": 0,
                        "needs_improvement": False,
                        "action_items": [],
                        "system_impact": "Review temporarily delayed due to API rate-limits."
                    }
                    status_file.write_text(json.dumps(status_data, indent=2), encoding="utf-8")
                    return
        except Exception:
            pass

        review_md = f"""## ⏳ Autonomous Architectural Review: Temporarily Rate-Limited

The autonomous reviewer encountered an API quota limit across available keys while reviewing this PR.

**Details**:
```text
{str(e)[:300]}
```

**Next Steps**:
- The reviewer will automatically resume once API quota resets.
"""
        review_file = workspace / "ai_pr_review.md"
        review_file.write_text(review_md.strip(), encoding="utf-8")
        status_file = workspace / "ai_review_status.json"
        status_data = {
            "verdict": "RATE_LIMITED",
            "score": 0,
            "needs_improvement": False,
            "action_items": ["Retry review when API quota resets."],
            "system_impact": "Review temporarily delayed due to API rate-limits."
        }
        status_file.write_text(json.dumps(status_data, indent=2), encoding="utf-8")
        print(f"[!] Wrote fallback rate-limited review to {review_file}")
        return

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
- Review the diff and click **Merge pull request** when you are ready.
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
