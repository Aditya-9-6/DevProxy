#!/usr/bin/env python3
"""
Autonomous CI Test & Build Auto-Fixer for DevProxy
Analyzes failing compiler, test, or lint errors, queries Google Gemini AI,
applies the fix, and verifies that tests pass before pushing.
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
Your mission is to fix failing Go build errors, compiler errors, data races, or broken unit tests in a Pull Request.

CRITICAL RULES:
1. PRESERVE EXISTING INTERFACES & EXPORTS: Never remove or omit existing structs, interfaces, methods, or helper functions that other files or packages depend on.
2. COMPILE-READY CODE: All files must be syntactically valid Go, with correct imports, correct types, and no undefined identifiers.
3. CONCURRENCY & PERFORMANCE: Maintain DevProxy's zero-allocation streaming patterns and race-free concurrency.
4. TARGETED FIXES: Fix the root cause of the error without rewriting unrelated features.
5. COMPLETE FILE CONTENT: When updating a file, provide the COMPLETE, FULL file content so it can replace the file directly.

CRITICAL OUTPUT FORMAT:
Respond ONLY with a single valid JSON object and nothing else (no conversational filler, no markdown wrappers outside JSON):
{
  "summary": "Clear explanation of what caused the CI failure and how it was resolved.",
  "files": [
    {
      "path": "relative/path/to/file.go",
      "content": "full updated file content"
    }
  ]
}
"""

def call_gemini(api_key: str, prompt: str, model: str = DEFAULT_MODEL) -> dict:
    """Calls Gemini REST API with fallback and retries across supported models."""
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

        print(f"[*] Querying model {current_model} for CI fix...", flush=True)
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
        # Keep relevant error lines
        out_lines.append(combined)

    combined_output = "\n".join(out_lines).strip()
    total_exit = 0 if not combined_output else 1
    return total_exit, combined_output

def extract_referenced_files(error_log: str, workspace: Path) -> list[str]:
    """Extracts Go file paths referenced in compiler or test error logs."""
    found = set()
    pattern = re.compile(r'([\w/\\.-]+\.go)(?::\d+)?')
    for match in pattern.finditer(error_log):
        rel_str = match.group(1).replace("\\", "/")
        p = workspace / rel_str
        if p.is_file():
            found.add(rel_str)
        else:
            # Check if match is relative to some subfolder
            for sub in workspace.rglob("*.go"):
                if sub.name == Path(rel_str).name:
                    try:
                        found.add(str(sub.relative_to(workspace)).replace("\\", "/"))
                    except ValueError:
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
    parser = argparse.ArgumentParser(description="DevProxy Autonomous CI Auto-Fixer")
    parser.add_argument("--pr-number", required=True, help="GitHub Pull Request Number")
    parser.add_argument("--workspace", default=".", help="Workspace root directory")
    parser.add_argument("--error-log-file", default="", help="Optional pre-captured error log file")
    args = parser.parse_args()

    api_key = os.environ.get("GEMINI_API_KEY", "").strip()
    if not api_key:
        print("MISSING_API_KEY: Environment variable GEMINI_API_KEY is not set.", file=sys.stderr)
        sys.exit(2)

    workspace = Path(args.workspace).resolve()
    print(f"[*] Starting Autonomous CI Fixer for PR #{args.pr_number} in {workspace}...")

    all_repaired_files = set()
    latest_summary = ""
    post_code = 1
    post_errors = ""

    max_iterations = 2
    for iteration in range(1, max_iterations + 1):
        print(f"\n=== Autonomous Repair Pass {iteration}/{max_iterations} ===", flush=True)

        # Step 1: Run diagnostics
        code, error_log = run_diagnostics(workspace)
        if code == 0 and not error_log:
            print(f"[OK] All local checks pass cleanly at pass {iteration}!")
            post_code = 0
            post_errors = ""
            break

        print(f"[*] Diagnostics identified {len(error_log)} chars of error output at pass {iteration}.")

        # Step 2: Extract referenced files & context
        referenced_files = extract_referenced_files(error_log, workspace)
        print(f"[*] Files mentioned in error output: {referenced_files}")

        file_contents = {}
        for rf in referenced_files[:10]:
            fp = workspace / rf
            if fp.is_file():
                try:
                    file_contents[rf] = fp.read_text(encoding="utf-8", errors="replace")
                except Exception as e:
                    print(f"[!] Could not read {rf}: {e}", file=sys.stderr)

        pr_diff = get_pr_diff(workspace)

        # Step 3: Construct prompt for Gemini
        prompt = f"""Pull Request #{args.pr_number} has failing CI checks / compiler errors (Repair Pass {iteration}/{max_iterations}).

=== CI DIAGNOSTIC ERROR LOG ===
{error_log[:18000]}

=== PULL REQUEST GIT DIFF ===
{pr_diff[:12000]}

=== REFERENCED SOURCE FILES CONTENT ===
{chr(10).join(f"--- File: {path} ---{chr(10)}{content}" for path, content in file_contents.items())}

Instructions:
1. Diagnose the root cause of every compilation error, undefined symbol, or failing test assertion above.
2. Note that if a symbol (like struct, const, or function) is already declared in another file in the same package (e.g. finding.go), DO NOT redeclare it in rules.go.
3. Ensure you preserve ALL existing exported types, functions, structs, and interfaces required across the package.
4. Provide the full replacement content for each file that needs to be fixed.
5. Output valid JSON adhering to the specified schema.
"""

        # Step 4: Request fix from Gemini
        result = call_gemini(api_key, prompt)
        latest_summary = result.get("summary", "Automated repair for CI test & compiler errors.")
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
            break

    # Write summary
    summary_file = workspace / "ci_fix_summary.md"
    summary_content = f"""## 🛠️ Autonomous CI Test & Build Fix Applied

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
