#!/usr/bin/env python3
"""
Autonomous AI Issue Approval & Gatekeeper Bot for DevProxy & GossipMesh.
Evaluates whether a proposed GitHub Issue (or PR) is genuinely necessary, architecturally sound,
and aligned with repository invariants.
- If APPROVED: Comments approval and triggers the PR maker agent (/solve).
- If CHANGES NEEDED: Communicates feedback and automatically updates/refines the issue specification.
- If UNNECESSARY: Comments rationale and closes the issue as not planned.
"""

import sys
import os
import json
import re
import time
import subprocess
import argparse
import urllib.request
import urllib.error
from pathlib import Path

DEFAULT_MODEL = "gemini-3.1-flash-lite"
FALLBACK_MODELS = [
    "gemini-3.1-flash-lite",
    "gemini-3.5-flash-lite",
    "gemini-3.5-flash",
    "gemini-flash-latest",
]

SYSTEM_PROMPT = """You are the Principal AI Architect and Gatekeeper.
Your job is to strictly evaluate whether a proposed GitHub Issue is genuinely NECESSARY, valuable, and architecturally sound, or if it is redundant, noise, anti-pattern, or unnecessary.

EVALUATION CRITERIA:
1. NECESSITY & VALUE:
   - Does this proposal deliver concrete, high-impact value (performance, reliability, core protocol compliance, resilience)?
   - Or is it redundant, trivial, duplicate of existing features, or adding useless complexity?
   - Score necessity from 1 to 100.
     * >= 80: High necessity and engineering value -> APPROVE
     * 55 - 79: Valuable concept, but needs architectural refinement -> NEEDS_CHANGES
     * < 55: Unnecessary, noise, or duplicate -> REJECT

2. ARCHITECTURAL RIGOR:
   - Must adhere to strict performance standards (zero-allocation in hot paths, thread-safety, modular design, clean separation).
   - Must have concrete acceptance criteria and testability.

3. COLLABORATIVE REFINEMENT:
   - If verdict is NEEDS_CHANGES, you MUST provide:
     * Specific constructive feedback.
     * A complete, refined, publication-ready Markdown issue body adhering to highest engineering standards.
     * An improved, precise title.

OUTPUT FORMAT:
Return ONLY valid JSON with this schema:
{
  "verdict": "APPROVE" | "NEEDS_CHANGES" | "REJECT",
  "necessity_score": <int 1-100>,
  "rationale": "<Detailed explanation of why this issue is or is not necessary>",
  "feedback": "<Specific feedback and guidance if changes needed, else empty>",
  "refined_title": "<Enhanced title if changes needed, else empty>",
  "refined_body": "<Complete refined issue markdown body if changes needed, else empty>"
}
"""

def load_gemini_keys() -> list[str]:
    keys = []
    pool = os.environ.get("GEMINI_KEY_POOL", "")
    for k in pool.split(","):
        if k.strip() and k.strip() not in keys:
            keys.append(k.strip())
    for var in ("GEMINI_ISSUE_KEY", "GEMINI_REVIEWER_KEY", "GEMINI_SOLVER_KEY", "GEMINI_API_KEY"):
        val = os.environ.get(var, "").strip()
        if val and val not in keys:
            keys.append(val)
    for check_dir in (Path("."), Path(__file__).resolve().parent, Path(__file__).resolve().parent.parent, Path.home()):
        kf = check_dir / ".gemini_keys"
        if kf.exists():
            try:
                for line in kf.read_text(encoding="utf-8").splitlines():
                    k = line.strip()
                    if k and not k.startswith("#") and k not in keys:
                        keys.append(k)
            except Exception:
                pass
    return keys

def extract_json_response(text: str) -> dict:
    text = text.strip()
    m = re.search(r"```(?:json)?\s*([\s\S]*?)\s*```", text)
    if m:
        text = m.group(1).strip()
    return json.loads(text)

def call_gemini(api_key: str, prompt: str, system_prompt: str = SYSTEM_PROMPT) -> dict:
    keys = [api_key] if api_key else []
    for k in load_gemini_keys():
        if k not in keys:
            keys.append(k)
    last_err = None
    for current_key in keys:
        for model in FALLBACK_MODELS:
            url = f"https://generativelanguage.googleapis.com/v1beta/models/{model}:generateContent?key={current_key}"
            payload = {
                "contents": [{"parts": [{"text": system_prompt}, {"text": prompt}]}],
                "generationConfig": {"temperature": 0.2, "responseMimeType": "application/json"}
            }
            req = urllib.request.Request(
                url, data=json.dumps(payload).encode("utf-8"),
                headers={"Content-Type": "application/json"}, method="POST"
            )
            try:
                with urllib.request.urlopen(req, timeout=15) as resp:
                    data = json.loads(resp.read().decode("utf-8"))
                    text = data["candidates"][0]["content"]["parts"][0]["text"].strip()
                    return extract_json_response(text)
            except urllib.error.HTTPError as e:
                last_err = e.read().decode("utf-8", errors="replace")
                if e.code in (400, 401, 403, 429):
                    break
                continue
            except Exception as e:
                last_err = str(e)
                continue
    raise RuntimeError(f"Gemini API exhausted. Last error: {last_err}")

def call_openrouter(api_key: str = "", prompt: str = "", system_prompt: str = SYSTEM_PROMPT) -> dict:
    key = api_key or os.environ.get("OPENROUTER_API_KEY", "").strip()
    if not key:
        raise ValueError("Missing OPENROUTER_API_KEY")
    url = "https://openrouter.ai/api/v1/chat/completions"
    models = ["google/gemini-2.0-flash-001", "meta-llama/llama-3.3-70b-instruct", "deepseek/deepseek-chat"]
    for model in models:
        payload = {
            "model": model,
            "messages": [{"role": "system", "content": system_prompt}, {"role": "user", "content": prompt}],
            "temperature": 0.2, "response_format": {"type": "json_object"}
        }
        req = urllib.request.Request(
            url, data=json.dumps(payload).encode("utf-8"),
            headers={"Authorization": f"Bearer {key}", "Content-Type": "application/json"}, method="POST"
        )
        try:
            with urllib.request.urlopen(req, timeout=30) as resp:
                data = json.loads(resp.read().decode("utf-8"))
                return extract_json_response(data["choices"][0]["message"]["content"])
        except Exception:
            continue
    raise RuntimeError("OpenRouter failed.")

def call_ai_model(prompt: str) -> dict:
    primary_gemini = os.environ.get("GEMINI_ISSUE_KEY", "").strip() or os.environ.get("GEMINI_API_KEY", "").strip()
    try:
        return call_gemini(primary_gemini, prompt)
    except Exception as e:
        print(f"[!] Gemini failed: {e}. Trying OpenRouter...", file=sys.stderr)
    try:
        return call_openrouter(prompt=prompt)
    except Exception as e:
        print(f"[!] OpenRouter failed: {e}", file=sys.stderr)
    raise RuntimeError("All AI models exhausted in issue approver.")

def run_cmd(cmd: list[str], cwd: Path = Path(".")) -> tuple[int, str, str]:
    res = subprocess.run(cmd, cwd=cwd, capture_output=True, text=True, encoding="utf-8", errors="replace")
    return res.returncode, res.stdout.strip(), res.stderr.strip()

def evaluate_issue(issue_number: str, workspace: Path, force: bool = False):
    print(f"[*] AI Gatekeeper evaluating Issue #{issue_number} in {workspace}...", flush=True)

    # 1. Fetch issue metadata
    code, stdout, stderr = run_cmd([
        "gh", "issue", "view", issue_number,
        "--json", "number,title,body,state,labels,author"
    ], cwd=workspace)
    if code != 0 or not stdout:
        print(f"[!] Failed to fetch Issue #{issue_number}: {stderr}", file=sys.stderr)
        return False

    issue = json.loads(stdout)
    if issue.get("state") != "OPEN":
        print(f"[*] Issue #{issue_number} is already closed ({issue.get('state')}). Skipping.")
        return True

    labels = [l.get("name", "") for l in issue.get("labels", [])]
    if not force and ("approved" in labels or "rejected" in labels):
        print(f"[*] Issue #{issue_number} already has terminal label {labels}. Skipping duplicate evaluation.")
        return True

    title = issue.get("title", "")
    body = issue.get("body", "")

    # 2. Fetch existing issues for context / deduplication
    _, issues_out, _ = run_cmd([
        "gh", "issue", "list", "--state", "all", "--limit", "30", "--json", "number,title"
    ], cwd=workspace)
    existing_titles = []
    try:
        existing_titles = [i["title"] for i in json.loads(issues_out) if str(i["number"]) != str(issue_number)]
    except Exception:
        pass

    prompt = f"""Evaluate this proposed GitHub Issue for necessity, value, and architectural rigor.

=== PROPOSED ISSUE #{issue_number} ===
Title: {title}
Body:
{body}

=== EXISTING REPOSITORY ISSUES (CHECK FOR DUPLICATION / REDUNDANCY) ===
{chr(10).join(f"- {t}" for t in existing_titles[:25])}

Instructions:
1. Determine if this issue is genuinely NECESSARY and high value, or if it should be rejected/closed.
2. If it is valuable but has minor flaws or needs tighter specs, choose NEEDS_CHANGES and provide the refined title and body.
3. If it is high quality and ready to implement, choose APPROVE.
4. Return JSON adhering to the specified schema.
"""

    print("[*] Dispatching issue evaluation to AI Gatekeeper...", flush=True)
    try:
        eval_result = call_ai_model(prompt)
    except Exception as e:
        print(f"[!] AI evaluation failed: {e}", file=sys.stderr)
        return False

    verdict = eval_result.get("verdict", "NEEDS_CHANGES").upper()
    score = eval_result.get("necessity_score", 70)
    rationale = eval_result.get("rationale", "")
    feedback = eval_result.get("feedback", "")
    refined_title = eval_result.get("refined_title", "").strip()
    refined_body = eval_result.get("refined_body", "").strip()

    print(f"[+] Evaluation Result for Issue #{issue_number}: Verdict={verdict}, Score={score}/100", flush=True)

    # Action 1: APPROVED
    if verdict == "APPROVE" or (verdict == "NEEDS_CHANGES" and score >= 85 and not refined_body):
        print(f"[*] Approving Issue #{issue_number} and triggering PR Solution Agent...", flush=True)
        run_cmd(["gh", "issue", "edit", issue_number, "--add-label", "approved,auto-solve"], cwd=workspace)

        comment = f"""### 🏛️ Architectural Gatekeeper: Issue Approved ✅

**Necessity Score**: **{score}/100**  
**Verdict**: **APPROVED FOR IMPLEMENTATION**

#### 📋 Architectural Evaluation
{rationale}

---
👉 **Triggering Autonomous PR Solution Agent...**
/solve
"""
        run_cmd(["gh", "issue", "comment", issue_number, "--body", comment], cwd=workspace)

        # Trigger PR solver workflow
        print(f"[*] Dispatching ai-issue-solver.yml for issue #{issue_number}...", flush=True)
        run_cmd(["gh", "workflow", "run", "ai-issue-solver.yml", "-f", f"issue_number={issue_number}"], cwd=workspace)
        return True

    # Action 2: NEEDS CHANGES (Autonomous Collaborative Refinement)
    elif verdict == "NEEDS_CHANGES":
        print(f"[*] Issue #{issue_number} requires refinement. Updating specification...", flush=True)
        
        # If the AI provided a refined body, update the issue directly!
        if refined_body:
            new_title = refined_title if refined_title else title
            run_cmd([
                "gh", "issue", "edit", issue_number,
                "--title", new_title,
                "--body", refined_body,
                "--add-label", "refined"
            ], cwd=workspace)
            print(f"[+] Successfully updated Issue #{issue_number} title and body.", flush=True)

        comment = f"""### 🏛️ Architectural Gatekeeper: Issue Refined & Upgraded 🔄

**Necessity Score**: **{score}/100**  
**Verdict**: **CHANGES APPLIED / SPECIFICATION REFINED**

#### 🔍 Architectural Review Feedback
{feedback or rationale}

#### 🛠️ Autonomous Refinement
*The Issue Maker Agent has incorporated the feedback directly into the issue description above. The ticket is now updated with strict acceptance criteria, performance invariants, and implementation scope.*

> 💡 Comment `/approve-issue` or `/solve` to proceed with automated PR creation.
"""
        run_cmd(["gh", "issue", "comment", issue_number, "--body", comment], cwd=workspace)
        return True

    # Action 3: REJECT / UNNECESSARY
    else:
        print(f"[*] Issue #{issue_number} rejected as unnecessary (Score: {score}/100). Closing...", flush=True)
        comment = f"""### 🛑 Architectural Gatekeeper: Issue Closed (Unnecessary / Not Planned)

**Necessity Score**: **{score}/100**  
**Verdict**: **REJECTED AS UNNECESSARY**

#### 🔍 Evaluation Rationale
{rationale}

*Closing this issue as not planned to maintain focus on core high-value roadmap items.*
"""
        run_cmd(["gh", "issue", "comment", issue_number, "--body", comment], cwd=workspace)
        run_cmd(["gh", "issue", "close", issue_number, "--reason", "not_planned"], cwd=workspace)
        run_cmd(["gh", "issue", "edit", issue_number, "--add-label", "rejected"], cwd=workspace)
        return True

def main():
    parser = argparse.ArgumentParser(description="AI Issue Approval & Gatekeeper Bot")
    parser.add_argument("--issue-number", required=True, help="GitHub Issue Number")
    parser.add_argument("--workspace", default=".", help="Workspace directory")
    parser.add_argument("--force", action="store_true", help="Force re-evaluation even if already labeled")
    args = parser.parse_args()

    workspace = Path(args.workspace).resolve()
    success = evaluate_issue(args.issue_number, workspace, force=args.force)
    sys.exit(0 if success else 1)

if __name__ == "__main__":
    main()
