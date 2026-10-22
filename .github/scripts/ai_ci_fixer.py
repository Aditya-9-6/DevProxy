#!/usr/bin/env python3
"""
Autonomous CI Test & Build Auto-Fixer for DevProxy
Analyzes failing compiler, test, or lint errors, queries Google Gemini AI,
applies the fix, and verifies that tests pass before pushing.
Also accepts AI Reviewer feedback to refactor spaghetti code and add missing tests.
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

DEFAULT_MODEL = "gemini-3.1-flash-lite"
FALLBACK_MODELS = [
    "gemini-3.1-flash-lite",
    "gemini-3.5-flash",
    "gemini-3.8-flash",
    "gemini-3.7-flash",
]

SYSTEM_PROMPT = """You are an expert autonomous Go systems engineer and compiler repair agent for DevProxy.
Your mission is to fix failing Go build errors, compiler errors, data races, broken unit tests, AND refactor spaghetti code based on AI Reviewer feedback.

CRITICAL RULES:
1. ANTI-SPAGHETTI & MODULARITY: Ensure functions are concise (<60 LOC), single-purpose, and decoupled. Refactor any tangled or duplicate logic identified by the reviewer.
2. PRESERVE EXISTING INTERFACES & EXPORTS: Never remove or omit existing structs, interfaces, methods, or helper functions that other files or packages depend on.
3. COMPILE-READY CODE: All files must be syntactically valid Go, with correct imports, correct types, and no undefined identifiers.
4. CONCURRENCY & PERFORMANCE: Maintain DevProxy's zero-allocation streaming patterns (sync.Pool) and race-free concurrency.
5. UNIT TEST GENERATION: Add comprehensive unit tests covering newly added functions, edge cases, and error branches.
6. COMPLETE FILE CONTENT: When updating a file, provide the COMPLETE, FULL file content so it can replace the file directly.

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

def call_gemini(api_key: str, prompt: str, fallback_key: str = "", model: str = DEFAULT_MODEL) -> dict:
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
                            {"text": SYSTEM_PROMPT},
                            {"text": prompt}
                        ]
                    }
                ],
                "generationConfig": {
                    "temperature": 0.1,
                    "responseMimeType": "application/json"
                }
            }

            print(f"[*] Querying model {current_model} for CI fix / refactoring...", flush=True)
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

    raise RuntimeError(f"Failed to obtain CI fix from Gemini API. Last error: {last_err}")

def run_diagnostics(workspace: Path) -> tuple[int, str]:
    """Runs compiler and test suite, returning exit code and combined error output."""
    print("[*] Running local diagnostic checks (gofmt, go vet, go test)...", flush=True)
    out_lines = []

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

    # Check go test
    test_res = subprocess.run(["go", "test", "-v", "./..."], cwd=workspace, capture_output=True, text=True)
    if test_res.returncode != 0:
        out_lines.append("=== GO TEST FAILURES ===")
        combined = (test_res.stdout + "\n" + test_res.stderr).strip()
        out_lines.append(combined)

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

    # Also include modified files in git status/diff
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
    parser = argparse.ArgumentParser(description="DevProxy Autonomous CI Auto-Fixer & Architectural Refactorer")
    parser.add_argument("--pr-number", required=True, help="GitHub Pull Request Number")
    parser.add_argument("--workspace", default=".", help="Workspace root directory")
    parser.add_argument("--error-log-file", default="", help="Optional pre-captured error log file")
    parser.add_argument("--review-feedback-file", default="", help="Optional reviewer feedback markdown file")
    args = parser.parse_args()

    primary_key = os.environ.get("GEMINI_SOLVER_KEY", "").strip() or os.environ.get("GEMINI_API_KEY", "").strip()
    fallback_key = os.environ.get("GEMINI_REVIEWER_KEY", "").strip() or os.environ.get("GEMINI_ISSUE_KEY", "").strip()

    if not primary_key:
        print("MISSING_API_KEY: Environment variable GEMINI_SOLVER_KEY or GEMINI_API_KEY is not set.", file=sys.stderr)
        sys.exit(2)

    workspace = Path(args.workspace).resolve()
    print(f"[*] Starting Autonomous CI Fixer for PR #{args.pr_number} in {workspace}...")

    review_feedback = ""
    if args.review_feedback_file and Path(args.review_feedback_file).is_file():
        review_feedback = Path(args.review_feedback_file).read_text(encoding="utf-8")
        print(f"[*] Loaded Reviewer Feedback ({len(review_feedback)} chars).")

    all_repaired_files = set()
    latest_summary = ""
    post_code = 1
    post_errors = ""

    max_iterations = 2
    for iteration in range(1, max_iterations + 1):
        print(f"\n=== Autonomous Repair Pass {iteration}/{max_iterations} ===", flush=True)

        # Step 1: Run diagnostics
        code, error_log = run_diagnostics(workspace)
        if code == 0 and not error_log and not review_feedback:
            print(f"[OK] All local checks pass cleanly at pass {iteration}!")
            post_code = 0
            post_errors = ""
            break

        print(f"[*] Diagnostics identified {len(error_log)} chars of error output at pass {iteration}.")

        # Step 2: Extract referenced files & context
        combined_context_text = error_log + "\n" + review_feedback
        referenced_files = extract_referenced_files(combined_context_text, workspace)
        print(f"[*] Files in repair scope: {referenced_files}")

        file_contents = {}
        for rf in referenced_files[:12]:
            fp = workspace / rf
            if fp.is_file():
                try:
                    file_contents[rf] = fp.read_text(encoding="utf-8", errors="replace")
                except Exception as e:
                    print(f"[!] Could not read {rf}: {e}", file=sys.stderr)

        pr_diff = get_pr_diff(workspace)

        # Step 3: Construct prompt for Gemini
        prompt = f"""Pull Request #{args.pr_number} requires autonomous refactoring / CI repair (Pass {iteration}/{max_iterations}).

=== ARCHITECTURAL REVIEW FEEDBACK & ANTI-SPAGHETTI DIRECTIVES ===
{review_feedback[:18000] if review_feedback else "No external reviewer feedback provided. Fix diagnostics."}

=== CI DIAGNOSTIC ERROR LOG ===
{error_log[:18000] if error_log else "Clean compiler output."}

=== PULL REQUEST GIT DIFF ===
{pr_diff[:12000]}

=== REFERENCED SOURCE FILES CONTENT ===
{chr(10).join(f"--- File: {path} ---{chr(10)}{content}" for path, content in file_contents.items())}

Instructions:
1. Address all architectural issues, anti-spaghetti recommendations, missing tests, and compiler/lint errors above.
2. Refactor any god functions (>60 lines) into clean, single-purpose helper functions.
3. Preserve all existing exported types, functions, structs, and interfaces needed across packages.
4. If missing unit tests were flagged, provide the full test file with table-driven tests.
5. Provide the full replacement content for each file that needs to be updated or created.
6. Output valid JSON adhering to the specified schema.
"""

        # Step 4: Request fix from Gemini
        result = call_gemini(primary_key, prompt, fallback_key=fallback_key)
        latest_summary = result.get("summary", "Automated repair for architectural feedback and CI errors.")
        files = result.get("files", [])

        if not files:
            print("[!] No file changes provided by AI.", file=sys.stderr)
            break

        print(f"[*] Applying {len(files)} fixed files...")
        for f in files:
            rel_path = f["path"].replace("\\", "/")
            content = f["content"]
            target = workspace / rel_path
            target.parent.mkdir(parents=True, exist_ok=True)
            target.write_text(content, encoding="utf-8")
            all_repaired_files.add(rel_path)
            print(f"    [+] Wrote fixed file: {rel_path}")

        # Step 5: Format & verify
        subprocess.run(["gofmt", "-w", "."], cwd=workspace)
        post_code, post_errors = run_diagnostics(workspace)
        if post_code == 0:
            print(f"[SUCCESS] Build and tests passed cleanly after pass {iteration}!")
            # Consume review feedback once applied
            review_feedback = ""
            break

    # Write summary
    summary_file = workspace / "ci_fix_summary.md"
    summary_content = f"""## 🛠️ Autonomous Architectural Repair & CI Fix Applied

**Target**: Pull Request #{args.pr_number}

### 📋 Fix Summary
{latest_summary}

### 📂 Files Repaired
{chr(10).join(f"- `{f}`" for f in sorted(list(all_repaired_files)))}

### 🧪 Diagnostic Verification
- Local build & test status after fix: **{'PASSED (Clean)' if post_code == 0 else 'WARNING (Some checks still reporting errors)'}**
{f"```text{chr(10)}{post_errors[:1500]}{chr(10)}```" if post_code != 0 else ""}
"""
    summary_file.write_text(summary_content, encoding="utf-8")
    print(f"[OK] Fix cycle completed. Final verification status: code {post_code}")

if __name__ == "__main__":
    main()
