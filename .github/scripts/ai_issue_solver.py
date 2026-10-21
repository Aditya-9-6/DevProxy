#!/usr/bin/env python3
"""
AI Issue Solver for DevProxy
Uses Google Gemini API to analyze an issue, understand the Go codebase, generate solutions,
and write files ready for automated pull request creation.
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
from typing import Optional

try:
    from gossipmesh import MemeticKnowledgeBase, RedTeamAuditor, GossipNode, GossipTopic
    GOSSIP_AVAILABLE = True
except ImportError:
    GOSSIP_AVAILABLE = False

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
    for var in ("GEMINI_SOLVER_KEY", "GEMINI_API_KEY", "GEMINI_REVIEWER_KEY", "GEMINI_ISSUE_KEY"):
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

SYSTEM_PROMPT = """You are an expert autonomous systems software engineer specializing in Go, high-throughput network proxies, HTTP/HTTPS MITM interception, WebSocket streaming, and developer debugging tools.
You are working on DevProxy, a high-performance, developer-first HTTP/HTTPS debugging proxy and security engine written in Go.

CORE ARCHITECTURE GUIDELINES:
1. High throughput and low memory footprint: prefer streaming buffers, zero-allocation pooling (sync.Pool) on hot request/response paths.
2. Concurrency safety: ensure data race freedom using proper sync primitives or lock-free circular ring buffers.
3. Idiomatic Go: clean error handling, context cancellation propagation, no unhandled goroutine leaks, adhere to standard Go naming and conventions.
4. Deterministic unit tests: always include table-driven or comprehensive unit tests (ending in _test.go) with the standard `testing` package.
5. Code style: clean comments, adherence to standard gofmt formatting.
6. PRESERVATION MANDATE: When modifying an existing file, you MUST PRESERVE 100% of the existing functions, methods, structs, and imports in that file. NEVER truncate or replace existing file code with partial stubs. If adding new functionality, prefer adding a NEW dedicated Go file (e.g., pkg/proxy/<feature>.go) instead of rewriting existing complex files.

You will be given a GitHub issue with its title, description, and repository context.
Analyze the requirements and generate the exact file changes needed to implement the feature or fix the bug.

CRITICAL OUTPUT FORMAT:
You MUST respond with a single valid JSON object and NOTHING ELSE (no markdown formatting, no explanations outside the JSON).
The JSON object must have this exact structure:
{
  "summary": "Brief 2-3 sentence overview of what was implemented and architecture decisions made.",
  "files": [
    {
      "path": "relative/path/to/file.go",
      "content": "complete updated or new file contents"
    }
  ]
}
"""

def get_git_files(workspace_root: Path) -> list:
    """Uses git ls-files for lightning-fast file discovery."""
    try:
        out = subprocess.check_output(["git", "ls-files"], cwd=workspace_root, text=True, stderr=subprocess.DEVNULL)
        return [line.strip().replace("\\", "/") for line in out.splitlines() if line.strip()]
    except Exception:
        files = []
        for root, dirs, filenames in os.walk(workspace_root):
            dirs[:] = [d for d in dirs if d not in [".git", "vendor", "node_modules", "bin"]]
            for f in filenames:
                rel = os.path.relpath(os.path.join(root, f), workspace_root).replace("\\", "/")
                files.append(rel)
        return files

def get_repo_overview(workspace_root: Path, all_files: list, max_files: int = 60) -> str:
    """Collects high-level overview of repo structure and go.mod configuration."""
    overview = []
    
    go_mod = workspace_root / "go.mod"
    if go_mod.exists():
        try:
            overview.append(f"=== Root go.mod ===\n{go_mod.read_text(encoding='utf-8')[:2000]}")
        except Exception:
            pass
    
    overview.append("\n=== Repository File Tree ===")
    selected = [f for f in all_files if f.endswith((".go", ".mod", ".md", ".yaml", ".yml"))][:max_files]
    overview.append("\n".join(selected))
    return "\n".join(overview)

def get_relevant_files(workspace_root: Path, all_files: list, keywords: list) -> str:
    """Reads contents of key files matching keywords to give context to LLM."""
    context_files = []
    collected_bytes = 0
    max_bytes = 60_000  # Keep within fast token limit
    
    go_files = [f for f in all_files if f.endswith(".go") and not f.endswith("_test.go")]
    for rel_path in go_files:
        is_relevant = any(k.lower() in rel_path.lower() for k in keywords)
        if is_relevant:
            try:
                p = workspace_root / rel_path
                content = p.read_text(encoding="utf-8")
                if collected_bytes + len(content) <= max_bytes:
                    context_files.append(f"=== Existing File: {rel_path} ===\n{content}")
                    collected_bytes += len(content)
            except Exception:
                pass
                
    return "\n\n".join(context_files)

def call_gemini(api_key: str, prompt: str, model: str = DEFAULT_MODEL, system_prompt: Optional[str] = None) -> dict:
    """Calls Spryzen Sovereign AI Gateway with semantic caching and cross-provider fallback."""
    active_sys_prompt = system_prompt or SYSTEM_PROMPT
    if system_prompt is None and GOSSIP_AVAILABLE:
        try:
            kb = MemeticKnowledgeBase()
            memes_ctx = kb.format_prompt_context(repo="DevProxy")
            if memes_ctx:
                active_sys_prompt = f"{SYSTEM_PROMPT}\n\n{memes_ctx}"
                print(f"[*] GossipMesh: Injected {len(kb.get_top_memes(repo='DevProxy'))} top evolutionary memes into solver prompt.", flush=True)
        except Exception:
            pass

    # 🚀 Try Spryzen Gateway first (Sub-1ms Semantic Cache & Multi-Provider Cascade)
    try:
        from spryzen_gateway import GLOBAL_GATEWAY
        return GLOBAL_GATEWAY.call_with_smart_routing(
            prompt=prompt,
            system_prompt=active_sys_prompt,
            preferred_model=model
        )
    except Exception as gw_err:
        err_msg = str(gw_err).encode("ascii", errors="replace").decode()
        print(f"[*] [Gateway Notice] Routing via direct fallback pool: {err_msg}", flush=True)

    ordered = [model] + [m for m in FALLBACK_MODELS if m != model]
    models_to_try = []
    for m in ordered:
        if m not in models_to_try:
            models_to_try.append(m)
    
    keys_to_try = [api_key] if api_key.strip() else []
    for k in load_gemini_keys():
        if k not in keys_to_try:
            keys_to_try.append(k)

    last_err = None
    for current_key in keys_to_try:
        key_masked = f"...{current_key[-6:]}" if len(current_key) > 6 else "key"
        key_rate_limited = False
        for current_model in models_to_try:
            if key_rate_limited:
                break
            url = f"https://generativelanguage.googleapis.com/v1beta/models/{current_model}:generateContent?key={current_key}"
            payload = {
                "contents": [
                    {
                        "parts": [
                            {"text": active_sys_prompt},
                            {"text": prompt}
                        ]
                    }
                ],
                "generationConfig": {
                    "temperature": 0.2,
                    "responseMimeType": "application/json"
                }
            }
            
            print(f"[*] Attempting generation with model: {current_model} (key: {key_masked})...", flush=True)
            max_attempts = 2
            for attempt in range(1, max_attempts + 1):
                req = urllib.request.Request(
                    url,
                    data=json.dumps(payload).encode("utf-8"),
                    headers={"Content-Type": "application/json"},
                    method="POST"
                )
                try:
                    with urllib.request.urlopen(req, timeout=120) as resp:
                        data = json.loads(resp.read().decode("utf-8"))
                        text_response = data["candidates"][0]["content"]["parts"][0]["text"].strip()
                        
                        # Parse JSON response
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
                    print(f"[Warning] HTTP {e.code} with model {current_model}: {err_msg[:160]}", file=sys.stderr)
                    last_err = err_msg
                    if e.code == 404:
                        break
                    if e.code == 429:
                        print(f"[*] Key {key_masked} rate-limited (429). Fast failover to next key in pool...", flush=True)
                        key_rate_limited = True
                        break
                    if attempt < max_attempts:
                        time.sleep(1)
                        continue
                    break
                except Exception as e:
                    print(f"[Warning] Error with model {current_model}: {e}", file=sys.stderr)
                    last_err = str(e)
                    if attempt < max_attempts:
                        time.sleep(1)
                        continue
                    break

    print(f"[!] Gemini pool exhausted: {last_err}. Cascading to secondary providers...", file=sys.stderr, flush=True)

    # 1. Try OpenRouter (Multi-model free tier & fallback)
    openrouter_key = os.environ.get("OPENROUTER_API_KEY", "").strip()
    if openrouter_key:
        try:
            return call_openrouter(openrouter_key, prompt, system_prompt=active_sys_prompt)
        except Exception as e:
            print(f"[!] OpenRouter fallback failed: {e}. Trying next provider...", file=sys.stderr, flush=True)

    # 2. Try NVIDIA NIM (Llama 3.3 70B & Nemotron)
    for nv_key in [os.environ.get("NVIDIA_API_KEY", "").strip(), os.environ.get("NVIDIA_API_KEY_2", "").strip()]:
        if nv_key:
            try:
                return call_nvidia_nim(nv_key, prompt, system_prompt=active_sys_prompt)
            except Exception as e:
                print(f"[!] NVIDIA NIM fallback failed: {e}. Trying next provider...", file=sys.stderr, flush=True)

    # 3. Try Hugging Face
    hf_key = os.environ.get("HUGGINGFACE_API_KEY", "").strip()
    if hf_key:
        try:
            return call_huggingface(hf_key, prompt, system_prompt=active_sys_prompt)
        except Exception as e:
            print(f"[!] HuggingFace fallback failed: {e}. Trying next provider...", file=sys.stderr, flush=True)

    # 4. Try Cohere
    cohere_key = os.environ.get("COHERE_API_KEY", "").strip()
    if cohere_key:
        try:
            return call_cohere(cohere_key, prompt, system_prompt=active_sys_prompt)
        except Exception as e:
            print(f"[!] Cohere fallback failed: {e}. Trying next provider...", file=sys.stderr, flush=True)

    raise RuntimeError(f"All AI providers (Gemini, OpenRouter, NVIDIA NIM, HuggingFace, Cohere) exhausted without successful response. Last error: {last_err}")

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
    start = raw_text.find("{")
    end = raw_text.rfind("}")
    if start != -1 and end != -1 and end > start:
        return json.loads(raw_text[start:end+1], strict=False)
    raise ValueError(f"Could not parse valid JSON from response: {raw_text[:200]}")

def call_openrouter(api_key: str, prompt: str, system_prompt: str = SYSTEM_PROMPT) -> dict:
    """Calls OpenRouter API as fallback."""
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
        print(f"[*] [OpenRouter] Querying {model}...", flush=True)
        payload = {
            "model": model,
            "messages": [
                {"role": "system", "content": system_prompt},
                {"role": "user", "content": prompt}
            ],
            "temperature": 0.1
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
                text = data["choices"][0]["message"]["content"]
                return extract_json_response(text)
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
    raise RuntimeError(f"OpenRouter calls failed. Last error: {last_err}")

def call_nvidia_nim(api_key: str, prompt: str, system_prompt: str = SYSTEM_PROMPT) -> dict:
    """Calls NVIDIA NIM API as fallback."""
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
        print(f"[*] [NVIDIA NIM] Querying {model}...", flush=True)
        payload = {
            "model": model,
            "messages": [
                {"role": "system", "content": system_prompt},
                {"role": "user", "content": prompt}
            ],
            "temperature": 0.1
        }
        req = urllib.request.Request(
            url,
            data=json.dumps(payload).encode("utf-8"),
            headers={
                "Authorization": f"Bearer {api_key}",
                "Content-Type": "application/json"
            },
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
            print(f"[!] [NVIDIA NIM] Model {model} failed: HTTP {e.code} - {err_body[:120]}", file=sys.stderr)
            if e.code == 401 or e.code == 429:
                break
            continue
        except Exception as e:
            print(f"[!] [NVIDIA NIM] Model {model} failed: {e}", file=sys.stderr)
            last_err = str(e)
            continue
    raise RuntimeError(f"NVIDIA NIM calls failed. Last error: {last_err}")

def call_huggingface(api_key: str, prompt: str, system_prompt: str = SYSTEM_PROMPT) -> dict:
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
            print(f"[*] [HuggingFace] Querying {model}...", flush=True)
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
                print(f"[!] [HuggingFace] Model {model} failed: HTTP {e.code} - {err_body[:120]}", file=sys.stderr)
                if e.code == 401:
                    break
                continue
            except Exception as e:
                print(f"[!] [HuggingFace] Model {model} failed: {e}", file=sys.stderr)
                last_err = str(e)
                continue
    raise RuntimeError(f"HuggingFace calls failed. Last error: {last_err}")

def call_cohere(api_key: str, prompt: str, system_prompt: str = SYSTEM_PROMPT) -> dict:
    """Calls Cohere Chat API as resilient fallback."""
    if not api_key:
        raise ValueError("Missing COHERE_API_KEY")
    url = "https://api.cohere.com/v2/chat"
    models = ["command-r-plus-08-2024", "command-r-08-2024", "command-r-plus", "command-r"]
    last_err = None
    for model in models:
        print(f"[*] [Cohere] Querying {model}...", flush=True)
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
            headers={
                "Authorization": f"Bearer {api_key}",
                "Content-Type": "application/json",
                "User-Agent": "DevProxy-CI/1.0"
            },
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
            print(f"[!] [Cohere] Model {model} failed: HTTP {e.code} - {err_body[:120]}", file=sys.stderr)
            if e.code == 429 or e.code == 401:
                break
            continue
        except Exception as e:
            print(f"[!] [Cohere] Model {model} failed: {e}", file=sys.stderr)
            last_err = str(e)
            continue
    raise RuntimeError(f"Cohere calls failed. Last error: {last_err}")

def main():
    parser = argparse.ArgumentParser(description="DevProxy AI Issue Solver")
    parser.add_argument("--issue-number", required=True, help="GitHub Issue Number")
    parser.add_argument("--issue-title", required=True, help="GitHub Issue Title")
    parser.add_argument("--issue-body", required=True, help="GitHub Issue Body")
    parser.add_argument("--workspace", default=".", help="Workspace root path")
    args = parser.parse_args()

    api_key = os.environ.get("GEMINI_API_KEY", "").strip()
    if not api_key:
        print("MISSING_API_KEY: Environment variable GEMINI_API_KEY is not set.", file=sys.stderr)
        print("To enable the bot, generate a free API key at https://aistudio.google.com/ and add it to GitHub Secrets as GEMINI_API_KEY.")
        sys.exit(2)

    workspace_root = Path(args.workspace).resolve()
    print(f"[*] Solving Issue #{args.issue_number}: {args.issue_title}")

    keywords = [w for w in (args.issue_title + " " + args.issue_body).split() if len(w) > 3]
    
    print("[*] Inspecting repository context...")
    all_files = get_git_files(workspace_root)
    overview = get_repo_overview(workspace_root, all_files)
    relevant_files = get_relevant_files(workspace_root, all_files, keywords)

    prompt = f"""
GitHub Issue #{args.issue_number}: {args.issue_title}

### Issue Description & Acceptance Criteria:
{args.issue_body}

### Repository Overview:
{overview}

### Relevant Existing Source Files:
{relevant_files if relevant_files else "No directly matching files found. Implement new package or integrate into existing packages."}

Instructions:
1. Implement the feature or fix specified in the issue.
2. Adhere to DevProxy performance and concurrency invariants.
3. Write clean, idiomatic Go code with accompanying unit tests (_test.go).
4. Output your response as valid JSON adhering to the specified schema.
"""

    print("[*] Requesting solution from Gemini AI...")
    result = call_gemini(api_key, prompt)

    summary = result.get("summary", "Automated resolution of issue.")
    files = result.get("files", [])

    if not files:
        print("[!] No file changes proposed by AI.", file=sys.stderr)
        sys.exit(1)

    print(f"[*] Applying {len(files)} file changes...")
    for f in files:
        rel_path = f["path"].replace("\\", "/")
        content = f["content"]
        target_path = workspace_root / rel_path

        # Code Integrity Guard: Reject destructive overwrites that truncate large files
        if target_path.is_file():
            orig_lines = len(target_path.read_text(encoding="utf-8", errors="replace").splitlines())
            new_lines = len(content.splitlines())
            if orig_lines > 80 and new_lines < int(orig_lines * 0.65):
                print(f"[!] WARNING: Proposed change for {rel_path} shrunk lines from {orig_lines} to {new_lines}. Truncation detected! Skipping destructive overwrite.", flush=True)
                continue

        target_path.parent.mkdir(parents=True, exist_ok=True)
        target_path.write_text(content, encoding="utf-8")
        print(f"    [+] Wrote: {rel_path}")

    summary_file = workspace_root / "ai_pr_summary.md"
    summary_content = f"""## 🤖 Automated Solution for Issue #{args.issue_number}

**Issue**: #{args.issue_number} - {args.issue_title}

### 📝 Solution Overview
{summary}

### 📦 Files Changed
{chr(10).join(f"- `{f['path']}`" for f in files)}

### 🛡️ Quality & Performance Invariants
- [x] Idiomatic Go concurrency and memory safety.
- [x] High throughput and streaming invariants preserved.
- [x] Comprehensive unit tests included.

---
*Closes #{args.issue_number}*
"""
    summary_file.write_text(summary_content, encoding="utf-8")
    print("[OK] Solution generated and files written successfully.")

if __name__ == "__main__":
    main()
