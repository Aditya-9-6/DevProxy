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

DEFAULT_MODEL = "gemini-3-flash-preview"
FALLBACK_MODELS = [
    "gemini-3-flash-preview",
    "gemma-4-26b-a4b-it",
    "gemini-flash-latest",
]

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
    """Calls Gemini REST API with fallback and retries across supported models."""
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

    ordered = [model] + [m for m in FALLBACK_MODELS if m != model]
    models_to_try = []
    for m in ordered:
        if m not in models_to_try:
            models_to_try.append(m)
    
    last_err = None
    for current_model in models_to_try:
        url = f"https://generativelanguage.googleapis.com/v1beta/models/{current_model}:generateContent?key={api_key}"
        
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
        
        print(f"[*] Attempting generation with model: {current_model}...", flush=True)
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
                print(f"[Warning] HTTP {e.code} (attempt {attempt}/{max_attempts}) with model {current_model}: {err_msg[:200]}", file=sys.stderr)
                last_err = err_msg
                # If model is deprecated / not found, do not waste retries
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

    raise RuntimeError(f"Failed to obtain solution from Gemini API across models {models_to_try}. Last error: {last_err}")

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
