#!/usr/bin/env python3
"""
Autonomous CI Test & Build Auto-Fixer for DevProxy
Analyzes failing compiler, test, or lint errors, queries AI models (Gemini, NVIDIA NIM, OpenRouter, Cohere),
resolves Git merge conflicts, applies the fix, and verifies that tests pass before pushing.
Also accepts AI Reviewer feedback to refactor code and add missing tests.
"""

import os
import sys
import json
import re
import argparse
import urllib.request
import urllib.error
import subprocess
import time
from pathlib import Path

DEFAULT_MODEL = "gemini-2.5-flash"
FALLBACK_MODELS = [
    "gemini-2.5-flash",
    "gemini-2.5-flash-lite",
    "gemini-3.1-flash-lite",
    "gemini-3.5-flash-lite",
    "gemini-3.5-flash",
    "gemini-3.7-flash",
    "gemini-3-flash-preview",
    "gemini-2.5-pro",
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

SYSTEM_PROMPT = """You are an expert autonomous Go systems engineer and compiler repair agent for DevProxy.
Your mission is to fix failing Go build errors, compiler errors, data races, broken unit tests, AND refactor spaghetti code based on AI Reviewer feedback.

CRITICAL RULES:
1. ANTI-SPAGHETTI & MODULARITY: Ensure functions are concise (<60 LOC), single-purpose, and decoupled. Refactor any tangled or duplicate logic identified by the reviewer.
2. PRESERVE EXISTING INTERFACES & EXPORTS: Never remove or omit existing structs, interfaces, methods, or helper functions that other files or packages depend on. NEVER truncate a file or replace it with a skeleton stub.
3. COMPILE-READY CODE: All files must be syntactically valid Go, with correct imports, correct types, and no undefined identifiers.
4. CONCURRENCY & PERFORMANCE: Maintain DevProxy's zero-allocation streaming patterns (sync.Pool) and race-free concurrency.
5. UNIT TEST GENERATION: Add comprehensive unit tests covering newly added functions, edge cases, and error branches.
6. COMPLETE FILE CONTENT: When updating a file, provide the COMPLETE, FULL file content so it can replace the file directly. Never write partial snippets or omit existing methods.

CRITICAL OUTPUT FORMAT:
Respond ONLY with a single valid JSON object and nothing else (no conversational filler, no markdown wrappers outside JSON):
{
  "summary": "Clear explanation of how the reviewer feedback was resolved and how the code was refactored.",
  "files": [
    {
      "path": "relative/path/to/file.go",
      "content": "full updated file content"
    }
  ]
}
"""

CONFLICT_SYSTEM_PROMPT = """You are an expert Git merge conflict resolver and Go systems architect for DevProxy.
Your mission is to resolve Git merge conflicts between `origin/main` and a feature pull request branch.

CRITICAL RULES:
1. PRESERVE INTENT FROM BOTH BRANCHES: Keep the enhancements, bug fixes, and additions introduced in main while preserving the new feature logic from the PR branch.
2. REMOVE ALL CONFLICT MARKERS: Completely remove <<<<<<< HEAD, =======, >>>>>>> markers.
3. COMPILE-READY GO CODE: Ensure imports, types, methods, and syntax are 100% valid Go code.
4. COMPLETE FILE CONTENT: Return the COMPLETE, entire resolved file content.

OUTPUT FORMAT:
Respond ONLY with a single valid JSON object:
{
  "summary": "Explanation of how conflicts were resolved.",
  "files": [
    {
      "path": "relative/path/to/file.go",
      "content": "full resolved file content"
    }
  ]
}
"""

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
    # Try finding the first '{' and last '}'
    first_brace = raw_text.find("{")
    last_brace = raw_text.rfind("}")
    if first_brace != -1 and last_brace != -1 and last_brace > first_brace:
        candidate = raw_text[first_brace:last_brace + 1]
        return json.loads(candidate, strict=False)
    raise ValueError(f"Could not parse valid JSON from model response: {raw_text[:200]}")

def call_gemini(api_key: str, prompt: str, system_prompt: str = SYSTEM_PROMPT, fallback_key: str = "", model: str = DEFAULT_MODEL) -> dict:
    """Calls Gemini REST API with fallback models and fallback API keys."""
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
                            {"text": system_prompt},
                            {"text": prompt}
                        ]
                    }
                ],
                "generationConfig": {
                    "temperature": 0.1,
                    "responseMimeType": "application/json"
                }
            }

            print(f"[*] [Gemini] Querying {current_model}...", flush=True)
            max_attempts = 3
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
                        return extract_json_response(text_response)
                except urllib.error.HTTPError as e:
                    err_msg = e.read().decode("utf-8", errors="replace")
                    last_err = err_msg
                    if e.code == 404:
                        break
                    if e.code == 429:
                        print(f"[*] [Gemini] Rate limited (429) on {current_model}. Fast failover...", flush=True)
                        break
                    if e.code in (500, 502, 503, 504) and attempt < max_attempts:
                        time.sleep(2)
                        continue
                    break
                except Exception as e:
                    last_err = str(e)
                    if attempt < max_attempts:
                        time.sleep(2 * attempt)
                        continue
                    break

    raise RuntimeError(f"Gemini API exhausted. Last error: {last_err}")

def call_openrouter(api_key: str, prompt: str, system_prompt: str = SYSTEM_PROMPT) -> dict:
    """Calls OpenRouter API as fallback."""
    if not api_key:
        raise ValueError("Missing OPENROUTER_API_KEY")
    url = "https://openrouter.ai/api/v1/chat/completions"
    models = ["deepseek/deepseek-chat", "meta-llama/llama-3.3-70b-instruct", "google/gemini-2.5-flash", "mistralai/mistral-large-2411"]
    last_err = None
    for model in models:
        print(f"[*] [OpenRouter] Querying {model}...", flush=True)
        payload = {
            "model": model,
            "messages": [
                {"role": "system", "content": system_prompt},
                {"role": "user", "content": prompt}
            ],
            "temperature": 0.1,
            "response_format": {"type": "json_object"}
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
    models = ["meta/llama-3.3-70b-instruct", "meta/llama-3.1-70b-instruct", "mistralai/mistral-large-2407"]
    last_err = None
    for model in models:
        print(f"[*] [NVIDIA NIM] Querying {model}...", flush=True)
        payload = {
            "model": model,
            "messages": [
                {"role": "system", "content": system_prompt},
                {"role": "user", "content": prompt}
            ],
            "temperature": 0.1,
            "response_format": {"type": "json_object"}
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
        except Exception as e:
            print(f"[!] [NVIDIA NIM] Model {model} failed: {e}", file=sys.stderr)
            last_err = str(e)
            continue
    raise RuntimeError(f"NVIDIA NIM calls failed. Last error: {last_err}")

def call_ai_model(prompt: str, system_prompt: str = SYSTEM_PROMPT) -> dict:
    """Universal multi-provider AI dispatcher with seamless failover."""
    # 1. Try Gemini
    primary_gemini = os.environ.get("GEMINI_SOLVER_KEY", "").strip() or os.environ.get("GEMINI_API_KEY", "").strip()
    fallback_gemini = os.environ.get("GEMINI_REVIEWER_KEY", "").strip() or os.environ.get("GEMINI_ISSUE_KEY", "").strip()
    try:
        return call_gemini(primary_gemini, prompt, system_prompt=system_prompt, fallback_key=fallback_gemini)
    except Exception as e:
        print(f"[!] Gemini attempt failed: {e}. Switching to secondary providers...", file=sys.stderr)

    # 2. Try NVIDIA NIM
    for nv_key in [os.environ.get("NVIDIA_API_KEY", "").strip(), os.environ.get("NVIDIA_API_KEY_2", "").strip()]:
        if nv_key:
            try:
                return call_nvidia_nim(nv_key, prompt, system_prompt=system_prompt)
            except Exception as e:
                print(f"[!] NVIDIA NIM key failed: {e}. Trying next provider...", file=sys.stderr)

    # 3. Try OpenRouter
    openrouter_key = os.environ.get("OPENROUTER_API_KEY", "").strip()
    if openrouter_key:
        try:
            return call_openrouter(openrouter_key, prompt, system_prompt=system_prompt)
        except Exception as e:
            print(f"[!] OpenRouter failed: {e}. Trying next provider...", file=sys.stderr)

    raise RuntimeError("All AI model providers (Gemini, NVIDIA NIM, OpenRouter) exhausted without successful response.")

def detect_and_resolve_conflicts(workspace: Path) -> list[str]:
    """Detects Git conflict markers and uses AI to resolve them cleanly."""
    conflicted_files = set()
    try:
        status_res = subprocess.run(["git", "status", "--porcelain"], cwd=workspace, capture_output=True, text=True)
        for line in status_res.stdout.splitlines():
            if line and line[:2] in ("UU", "AA", "UD", "DU", "AU", "UA", "DD"):
                conflicted_files.add(line[3:].strip().replace("\\", "/"))
    except Exception:
        pass

    for p in workspace.rglob("*.go"):
        try:
            content = p.read_text(encoding="utf-8", errors="ignore")
            if "<<<<<<< HEAD" in content or ("=======" in content and ">>>>>>>" in content):
                conflicted_files.add(str(p.relative_to(workspace)).replace("\\", "/"))
        except Exception:
            pass

    resolved = []
    if not conflicted_files:
        return resolved

    print(f"[*] ⚠️ Detected {len(conflicted_files)} conflicted file(s): {sorted(list(conflicted_files))}", flush=True)
    for rel_path in sorted(list(conflicted_files)):
        target_file = workspace / rel_path
        if not target_file.is_file():
            print(f"[*] Handling deleted/missing conflicted file {rel_path} by restoring from origin/main...", flush=True)
            subprocess.run(["git", "checkout", "origin/main", "--", rel_path], cwd=workspace)
            subprocess.run(["git", "add", rel_path], cwd=workspace)
            resolved.append(rel_path)
            continue
        conflicted_text = target_file.read_text(encoding="utf-8", errors="replace")
        if "<<<<<<<" not in conflicted_text and ">>>>>>>" not in conflicted_text:
            print(f"[*] Resolving non-marker conflict in {rel_path} by restoring from origin/main...", flush=True)
            subprocess.run(["git", "checkout", "origin/main", "--", rel_path], cwd=workspace)
            subprocess.run(["git", "add", rel_path], cwd=workspace)
            resolved.append(rel_path)
            continue

        print(f"[*] Resolving Git merge conflicts in {rel_path} via AI...", flush=True)
        prompt = f"""Resolve the Git merge conflicts in `{rel_path}`.

CRITICAL INSTRUCTIONS:
1. Merge the changes from HEAD (the PR branch) and the incoming base branch (origin/main).
2. Retain all functionality, imports, struct fields, and methods from BOTH branches.
3. Completely remove all conflict markers (`<<<<<<< HEAD`, `=======`, `>>>>>>> ...`).
4. Ensure the resulting file is 100% syntactically valid Go.
5. Return the COMPLETE, entire resolved file content in the JSON schema.

=== CONFLICTED FILE: {rel_path} ===
{conflicted_text}
"""
        try:
            res = call_ai_model(prompt, system_prompt=CONFLICT_SYSTEM_PROMPT)
            files = res.get("files", [])
            for f in files:
                if f.get("content"):
                    dest = workspace / f["path"].replace("\\", "/")
                    dest.write_text(f["content"], encoding="utf-8")
                    subprocess.run(["git", "add", f["path"].replace("\\", "/")], cwd=workspace)
                    resolved.append(rel_path)
                    print(f"    [+] Successfully resolved conflict in {rel_path}", flush=True)
        except Exception as e:
            print(f"[!] Conflict resolution failed for {rel_path}: {e}", file=sys.stderr)

    # Resolve any remaining unmerged paths
    try:
        remaining_res = subprocess.run(["git", "diff", "--name-only", "--diff-filter=U"], cwd=workspace, capture_output=True, text=True)
        remaining = [l.strip() for l in remaining_res.stdout.splitlines() if l.strip()]
        for rem in remaining:
            print(f"[*] Fallback: resolving remaining unmerged file {rem} using origin/main...", flush=True)
            subprocess.run(["git", "checkout", "origin/main", "--", rem], cwd=workspace)
            subprocess.run(["git", "add", rem], cwd=workspace)
            resolved.append(rem)
    except Exception:
        pass

    if resolved:
        subprocess.run(["git", "add", "-A"], cwd=workspace)
        merge_head = workspace / ".git" / "MERGE_HEAD"
        if merge_head.exists():
            subprocess.run([
                "git", "commit", "-m", "fix(merge): resolve merge conflicts with origin/main",
                "-m", "Co-authored-by: Aditya Dahale <aditya-9-6@users.noreply.github.com>"
            ], cwd=workspace)
            print("[*] Completed merge commit after conflict resolution.", flush=True)

    return resolved

def fetch_remote_ci_errors(pr_number: str, workspace: Path) -> str:
    """Fetches failed GitHub Actions logs for this PR if available."""
    try:
        head_ref = ""
        pr_view = subprocess.run(["gh", "pr", "view", pr_number, "--json", "headRefName"], cwd=workspace, capture_output=True, text=True)
        if pr_view.returncode == 0:
            head_ref = json.loads(pr_view.stdout).get("headRefName", "")

        cmd = ["gh", "run", "list", "--limit", "10", "--json", "databaseId,conclusion,name,headBranch"]
        if head_ref:
            cmd.extend(["-b", head_ref])

        res = subprocess.run(cmd, cwd=workspace, capture_output=True, text=True)
        if res.returncode == 0:
            data = json.loads(res.stdout)
            for run in data:
                if run.get("conclusion") in ("failure", "timed_out", "startup_failure"):
                    run_id = str(run.get("databaseId"))
                    print(f"[*] Found failed GitHub Actions run {run_id} ({run.get('name')}). Fetching failed logs...", flush=True)
                    log_res = subprocess.run(
                        ["gh", "run", "view", "--log-failed", run_id],
                        cwd=workspace, capture_output=True, text=True
                    )
                    if log_res.stdout.strip():
                        return f"=== GITHUB ACTIONS FAILED RUN ({run.get('name')} #{run_id}) ===\n" + log_res.stdout.strip()[:15000]
    except Exception as e:
        print(f"[*] Note: Could not fetch remote CI logs via gh CLI: {e}", flush=True)
    return ""

def run_diagnostics(workspace: Path, pr_number: str = "") -> tuple[int, str]:
    """Runs compiler and test suite, returning exit code and combined error output."""
    print("[*] Running local diagnostic checks (go mod tidy, gofmt, go vet, go build, go test)...", flush=True)
    out_lines = []

    # Run go mod tidy
    subprocess.run(["go", "mod", "tidy"], cwd=workspace, capture_output=True, text=True)

    # Check gofmt
    fmt_res = subprocess.run(["gofmt", "-l", "."], cwd=workspace, capture_output=True, text=True)
    if fmt_res.stdout.strip():
        out_lines.append("=== GOFMT FORMATTING ERRORS ===")
        out_lines.append(f"Unformatted files:\n{fmt_res.stdout.strip()}\n")

    # Check go vet
    vet_res = subprocess.run(["go", "vet", "./..."], cwd=workspace, capture_output=True, text=True)
    if vet_res.returncode != 0:
        out_lines.append("=== GO VET COMPILATION / STATIC ANALYSIS ERRORS ===")
        out_lines.append(vet_res.stderr.strip() or vet_res.stdout.strip())
        out_lines.append("")

    # Check go build
    build_res = subprocess.run(["go", "build", "./..."], cwd=workspace, capture_output=True, text=True)
    if build_res.returncode != 0:
        out_lines.append("=== GO BUILD COMPILATION ERRORS ===")
        out_lines.append(build_res.stderr.strip() or build_res.stdout.strip())
        out_lines.append("")

    # Check go test
    test_res = subprocess.run(["go", "test", "-v", "./..."], cwd=workspace, capture_output=True, text=True)
    if test_res.returncode != 0:
        out_lines.append("=== GO TEST FAILURES ===")
        combined = (test_res.stdout + "\n" + test_res.stderr).strip()
        out_lines.append(combined)

    # Check remote GitHub CI failures if available
    if pr_number:
        remote_ci = fetch_remote_ci_errors(pr_number, workspace)
        if remote_ci:
            out_lines.append(remote_ci)

    combined_output = "\n".join(out_lines).strip()
    total_exit = 0 if not combined_output else 1
    return total_exit, combined_output

def extract_referenced_files(error_log: str, workspace: Path) -> list[str]:
    """Extracts Go file paths referenced in logs or diff."""
    found = set()
    pattern = re.compile(r'([\w/\\.-]+\.go)(?::\d+)?')
    for match in pattern.finditer(error_log):
        rel_str = match.group(1).replace("\\", "/")
        p = workspace / rel_str
        if p.is_file():
            found.add(rel_str)
        else:
            for sub in workspace.rglob("*.go"):
                if sub.name == Path(rel_str).name:
                    try:
                        found.add(str(sub.relative_to(workspace)).replace("\\", "/"))
                    except ValueError:
                        pass

    try:
        diff_names = subprocess.run(["git", "diff", "--name-only", "origin/main...HEAD"], cwd=workspace, capture_output=True, text=True)
        for line in diff_names.stdout.splitlines():
            line = line.strip().replace("\\", "/")
            if line.endswith(".go") and (workspace / line).is_file():
                found.add(line)
    except Exception:
        pass

    return sorted(list(found))

def get_pr_diff(workspace: Path) -> str:
    """Gets git diff against origin/main or HEAD~1."""
    try:
        res = subprocess.run(["git", "diff", "origin/main...HEAD"], cwd=workspace, capture_output=True, text=True)
        if res.stdout.strip():
            return res.stdout.strip()[:15000]
    except Exception:
        pass
    try:
        res = subprocess.run(["git", "diff", "HEAD~1"], cwd=workspace, capture_output=True, text=True)
        if res.stdout.strip():
            return res.stdout.strip()[:15000]
    except Exception:
        pass
    return "No git diff available."

def main():
    parser = argparse.ArgumentParser(description="DevProxy Autonomous CI Auto-Fixer & Merge Conflict Resolver")
    parser.add_argument("--pr-number", required=True, help="GitHub Pull Request Number")
    parser.add_argument("--workspace", default=".", help="Workspace root directory")
    parser.add_argument("--error-log-file", default="", help="Optional pre-captured error log file")
    parser.add_argument("--review-feedback-file", default="", help="Optional reviewer feedback markdown file")
    parser.add_argument("--max-iterations", type=int, default=6, help="Maximum self-healing loop passes (default: 6)")
    args = parser.parse_args()

    workspace = Path(args.workspace).resolve()
    print(f"[*] Starting Autonomous CI Fixer & Conflict Resolver for PR #{args.pr_number} in {workspace}...")

    # Phase 0: Resolve any merge conflicts first
    conflicts_resolved = detect_and_resolve_conflicts(workspace)
    if conflicts_resolved:
        print(f"[OK] Automatically resolved merge conflicts in: {conflicts_resolved}", flush=True)

    review_feedback = ""
    if args.review_feedback_file and Path(args.review_feedback_file).is_file():
        review_feedback = Path(args.review_feedback_file).read_text(encoding="utf-8")
        print(f"[*] Loaded Reviewer Feedback ({len(review_feedback)} chars).")

    all_repaired_files = set(conflicts_resolved)
    latest_summary = "Resolved Git merge conflicts." if conflicts_resolved else ""
    post_code = 1
    initial_had_errors = False

    max_iterations = args.max_iterations
    for iteration in range(1, max_iterations + 1):
        print(f"\n=======================================================", flush=True)
        print(f"[*] Autonomous Repair Pass {iteration}/{max_iterations}", flush=True)
        print(f"=======================================================", flush=True)

        # Step 1: Run diagnostics
        code, error_log = run_diagnostics(workspace, pr_number=args.pr_number)
        if code != 0 or error_log:
            initial_had_errors = True

        if code == 0 and not error_log:
            if not review_feedback or iteration > 1:
                print(f"\n[🎉 SUCCESS] All checks passed 100% GREEN at pass {iteration}! 0 errors, 0 test failures.", flush=True)
                post_code = 0
                break
            else:
                print(f"[*] Diagnostics green, but architectural review feedback must be resolved on Pass 1...", flush=True)

        if error_log:
            print(f"[*] Diagnostics identified failing checks at pass {iteration}:\n{error_log[:500]}...", flush=True)

        # Step 2: Extract referenced files & context
        combined_context_text = (error_log or "") + "\n" + (review_feedback or "")
        referenced_files = extract_referenced_files(combined_context_text, workspace)
        print(f"[*] Files in repair scope: {referenced_files}", flush=True)

        file_contents = {}
        for rf in referenced_files[:12]:
            fp = workspace / rf
            if fp.is_file():
                try:
                    file_contents[rf] = fp.read_text(encoding="utf-8", errors="replace")
                except Exception as e:
                    print(f"[!] Could not read {rf}: {e}", file=sys.stderr)

        pr_diff = get_pr_diff(workspace)

        # Step 3: Construct prompt
        if not error_log and review_feedback:
            prompt_header = f"""Pull Request #{args.pr_number} compiles and tests pass, but the Architectural Reviewer flagged anti-patterns / action items (Pass {iteration}/{max_iterations}).
Your goal: Refactor the code to resolve all architectural review action items while keeping all tests 100% GREEN."""
        else:
            prompt_header = f"""Pull Request #{args.pr_number} has failing Go tests or compiler errors (Pass {iteration}/{max_iterations}).
Your goal: Fix all compiler errors and broken tests so that 'go test -v ./...' passes 100% GREEN."""

        prompt = f"""{prompt_header}

=== CI DIAGNOSTIC ERROR LOG ===
{error_log[:15000] if error_log else "None (compilation clean). Focus on architectural review feedback."}

{f"=== ARCHITECTURAL REVIEW FEEDBACK ==={chr(10)}{review_feedback[:10000]}" if review_feedback else ""}

=== CURRENT GIT DIFF ===
{pr_diff[:10000]}

=== REFERENCED SOURCE FILES (PRESERVE ALL EXISTING CODE) ===
{chr(10).join(f"--- File: {path} ---{chr(10)}{content}" for path, content in file_contents.items())}

Instructions:
1. Examine the exact go compiler errors and failing unit tests above.
2. Fix broken logic, type mismatches, missing imports, or missing methods.
3. PRESERVATION MANDATE: You MUST preserve all existing structs, methods, interfaces, and packages. NEVER delete or truncate existing functionality.
4. Provide the full replacement content for each file that needs to be updated.
5. Output valid JSON adhering to the specified schema.
"""

        # Step 4: Request fix from multi-provider AI engine
        try:
            result = call_ai_model(prompt, system_prompt=SYSTEM_PROMPT)
        except Exception as e:
            print(f"[!] AI call failed on pass {iteration}: {e}", file=sys.stderr)
            time.sleep(5)
            continue

        latest_summary = result.get("summary", "Automated repair for Go CI diagnostics.")
        files = result.get("files", [])

        if not files:
            print("[!] No file changes provided by AI.", file=sys.stderr)
            break

        print(f"[*] Applying {len(files)} fixed file(s)...", flush=True)
        for f in files:
            rel_path = f["path"].replace("\\", "/")
            content = f["content"]
            target = workspace / rel_path

            # Code Integrity Guard: Prevent destructive file truncation
            if target.is_file():
                orig_lines = len(target.read_text(encoding="utf-8", errors="replace").splitlines())
                new_lines = len(content.splitlines())
                if orig_lines > 80 and new_lines < int(orig_lines * 0.65):
                    print(f"[!] WARNING: Proposed fix for {rel_path} shrunk lines from {orig_lines} to {new_lines}. Truncation detected! Skipping destructive overwrite.", flush=True)
                    continue

            target.parent.mkdir(parents=True, exist_ok=True)
            target.write_text(content, encoding="utf-8")
            all_repaired_files.add(rel_path)
            print(f"    [+] Wrote fixed file: {rel_path}", flush=True)

        # Step 5: Format & verify
        subprocess.run(["go", "mod", "tidy"], cwd=workspace)
        subprocess.run(["gofmt", "-w", "."], cwd=workspace)

    final_code, final_log = run_diagnostics(workspace, pr_number=args.pr_number)
    post_code = final_code

    # Prevent false "100% GREEN" if no repairs were produced while initial errors were present
    if initial_had_errors and not all_repaired_files and post_code != 0:
        status_label = "⚠️ **CHECKS STILL REPORTING ERRORS (No Auto-Fix Produced)**"
    elif post_code == 0:
        status_label = "✅ **ALL TESTS 100% GREEN (Passed)**"
    else:
        status_label = "⚠️ **CHECKS STILL REPORTING ERRORS**"

    summary_file = workspace / "ci_fix_summary.md"
    summary_content = f"""## 🛠️ Autonomous CI Test Fixer Report (DevProxy Go)

**Target**: Pull Request #{args.pr_number}
**Status**: {status_label}

### 📋 Resolution Summary
{latest_summary or "CI diagnostics analyzed."}

### 📂 Files Modified
{chr(10).join(f"- `{f}`" for f in sorted(list(all_repaired_files))) if all_repaired_files else "None"}

### 🧪 Diagnostic Verification
- Local build & test exit code: **{post_code}**
{f"```text{chr(10)}{final_log[:1500]}{chr(10)}```" if post_code != 0 else "```text\nAll Go packages passed go vet and go test cleanly.\n```"}
"""
    summary_file.write_text(summary_content, encoding="utf-8")
    print(f"\n[OK] Fix cycle completed. Final verification status: exit code {post_code}")

    if post_code == 0:
        print("[🎉 100% GREEN] CI suite passed cleanly.")
        sys.exit(0)
    else:
        print(f"[!] Tests still failing after repair passes.")
        sys.exit(1)

if __name__ == "__main__":
    main()
