#!/usr/bin/env python3
"""
DevProxy Architecture Advancement Issue Generator Engine (High-Quality & Deduplication Gate Edition)
Autonomous issue generator bot for continuous open-source contribution & architectural advancement.
Strictly checks whether a proposed issue is already reported or solved before submitting.
Rejects trivial tasks, duplicates, and existing codebase features.
"""

import json
import os
import re
import subprocess
import sys
import time
import urllib.request
import urllib.error
from datetime import datetime, timezone
from pathlib import Path

# Optional GossipMesh integration
try:
    from gossipmesh import MemeticKnowledgeBase, GossipNode, GossipTopic
    GOSSIP_AVAILABLE = True
except ImportError:
    GOSSIP_AVAILABLE = False

DEFAULT_MODEL = "gemini-flash-lite-latest"
FALLBACK_MODELS = ["gemini-flash-lite-latest", "gemini-flash-latest", "gemini-pro-latest"]

# Curated catalog of verified, high-impact, UNIMPLEMENTED open-source architecture advancements for DevProxy (Go)
CATALOG = [
    {
        "title": "feat(quic): Add HTTP/3 QUIC connection listener & packet demuxer",
        "area": "area/proxy",
        "difficulty": "enhancement",
        "spec": "Implement an experimental HTTP/3 UDP listener via quic-go to accept QUIC datagrams, demux streams, and bridge incoming HTTP/3 client requests to DevProxy's ringbuffer interception pipeline with zero packet copy.",
        "target_files": ["pkg/proxy/quic.go", "pkg/proxy/proxy.go"]
    },
    {
        "title": "feat(wasm): Add WebAssembly (Wasm) request/response filter plugin runtime using wazero",
        "area": "area/proxy",
        "difficulty": "enhancement",
        "spec": "Integrate the pure-Go wazero WebAssembly runtime to allow users to load custom compiled .wasm interceptor filters that modify HTTP headers and payloads in-flight with zero CGO dependencies and memory-isolated execution.",
        "target_files": ["pkg/wasm/runtime.go", "pkg/proxy/proxy.go"]
    },
    {
        "title": "feat(dns): Add DNS-over-HTTPS (DoH) upstream resolver with caching & TTL eviction",
        "area": "area/proxy",
        "difficulty": "enhancement",
        "spec": "Implement a concurrent RFC 8484 DNS-over-HTTPS client with in-memory lock-free LRU cache and automatic TTL expiration to securely resolve upstream proxy targets, bypassing local DNS poisoning and split-horizon leaks.",
        "target_files": ["pkg/dns/doh.go", "pkg/proxy/upstream.go"]
    },
    {
        "title": "feat(graphql): Add GraphQL query depth & cyclic recursion limiter in WAF",
        "area": "area/security",
        "difficulty": "enhancement",
        "spec": "Parse incoming POST application/json GraphQL documents using an AST visitor to calculate maximum selection set depth and cyclic fragment recursion, immediately returning HTTP 400 when exceeding depth thresholds.",
        "target_files": ["pkg/analysis/graphql_depth.go", "pkg/analysis/rules.go"]
    },
    {
        "title": "feat(storage): Add streaming zstd compression for session log archives",
        "area": "area/replay",
        "difficulty": "enhancement",
        "spec": "Implement zero-allocation streaming zstandard compression (via klauspost/compress/zstd) for HAR and raw traffic event dumps, reducing disk storage footprint by over 80% without stalling proxy worker threads.",
        "target_files": ["pkg/storage/zstd.go", "pkg/storage/har.go"]
    },
    {
        "title": "feat(metrics): Add Prometheus OTLP exporter for real-time proxy metrics",
        "area": "area/proxy",
        "difficulty": "enhancement",
        "spec": "Implement a Prometheus exporter endpoint (/metrics) exposing real-time connection counts, active goroutines, bytes sent/received, ringbuffer drops, and upstream response latency percentiles.",
        "target_files": ["pkg/metrics/exporter.go", "pkg/dashboard/server.go"]
    },
    {
        "title": "feat(ratelimit): Implement token-bucket and sliding-window rate limiter middleware",
        "area": "area/proxy",
        "difficulty": "enhancement",
        "spec": "Implement a thread-safe in-memory token bucket rate limiter with sliding-window log tracking per client IP, allowing configurable requests-per-second thresholds and automatic HTTP 429 response injection.",
        "target_files": ["pkg/ratelimit/limiter.go", "pkg/proxy/proxy.go"]
    },
    {
        "title": "feat(grpc): Add bidirectional gRPC mock reflection engine with protobuf descriptors",
        "area": "area/proxy",
        "difficulty": "enhancement",
        "spec": "Implement dynamic gRPC server reflection and mock payload generation from raw .proto or file descriptor sets, enabling developers to mock streaming gRPC endpoints without recompiling protobuf stubs.",
        "target_files": ["pkg/mock/grpc.go", "pkg/proxy/grpc.go"]
    }
]

STOPWORDS = {
    "a", "an", "the", "and", "or", "in", "on", "at", "to", "for", "of", "with",
    "by", "from", "into", "using", "via", "is", "it", "as", "be", "this", "that",
    "add", "feat", "implement", "create", "support", "area", "enhancement", "fix"
}

def extract_keywords(text: str) -> set[str]:
    """Extracts significant lowercase alphanumeric tokens for keyword-based comparison."""
    tokens = re.findall(r"[a-zA-Z0-9_\-]+", text.lower())
    clean = set()
    for t in tokens:
        t = t.strip("-_")
        if len(t) > 2 and t not in STOPWORDS:
            clean.add(t)
    return clean

def get_repository_intel(workspace: Path) -> dict:
    """
    Fetches comprehensive repository intelligence from GitHub and local codebase:
    - All open and closed issues
    - All open, closed, and merged pull requests
    - Local file manifest (existing packages and source files)
    """
    intel = {
        "issues": [],
        "prs": [],
        "open_issues": [],
        "closed_issues": [],
        "open_prs": [],
        "merged_prs": [],
        "all_titles": set(),
        "code_files": set(),
    }

    # 1. Fetch GitHub issues
    try:
        cmd = ["gh", "issue", "list", "--state", "all", "--limit", "200", "--json", "number,title,state,createdAt"]
        res = subprocess.run(cmd, cwd=workspace, capture_output=True, text=True, check=True)
        issues = json.loads(res.stdout) if res.stdout else []
        intel["issues"] = issues
        for i in issues:
            title = i.get("title", "").strip()
            state = i.get("state", "").upper()
            intel["all_titles"].add(title)
            if state == "OPEN":
                intel["open_issues"].append(i)
            else:
                intel["closed_issues"].append(i)
    except Exception as e:
        print(f"[WARN] Failed to fetch issues via gh CLI: {e}")

    # 2. Fetch GitHub PRs
    try:
        cmd = ["gh", "pr", "list", "--state", "all", "--limit", "200", "--json", "number,title,state,headRefName"]
        res = subprocess.run(cmd, cwd=workspace, capture_output=True, text=True, check=True)
        prs = json.loads(res.stdout) if res.stdout else []
        intel["prs"] = prs
        for p in prs:
            title = p.get("title", "").strip()
            state = p.get("state", "").upper()
            intel["all_titles"].add(title)
            if state == "OPEN":
                intel["open_prs"].append(p)
            elif state == "MERGED":
                intel["merged_prs"].append(p)
    except Exception as e:
        print(f"[WARN] Failed to fetch PRs via gh CLI: {e}")

    # 3. Local codebase manifest
    try:
        for p in workspace.rglob("*.go"):
            if ".git" not in p.parts:
                rel = str(p.relative_to(workspace)).replace("\\", "/")
                intel["code_files"].add(rel)
    except Exception:
        pass

    return intel

def is_already_reported_or_solved(item: dict, intel: dict, workspace: Path) -> tuple[bool, str]:
    """
    Checks whether a proposed issue specification is:
    1. Already reported (matches an open issue or active PR)
    2. Already solved (matches a closed issue, merged PR, or already implemented in codebase)
    Returns: (is_duplicate: bool, reason: str)
    """
    title = item.get("title", "").strip()
    spec = item.get("spec", "").strip()
    target_files = item.get("target_files", [])
    cand_tokens = extract_keywords(title) | extract_keywords(spec)

    # Check 1: Exact title match across all issues and PRs
    clean_cand_title = re.sub(r"\s+", " ", title.lower().strip())
    for existing in intel.get("issues", []) + intel.get("prs", []):
        ext_title = existing.get("title", "").strip()
        clean_ext_title = re.sub(r"\s+", " ", ext_title.lower().strip())
        if clean_cand_title == clean_ext_title:
            state = existing.get("state", "UNKNOWN")
            num = existing.get("number", "?")
            return True, f"Exact title matches existing issue/PR #{num} (State: {state}): '{ext_title}'"

    # Check 2: Technical keyword overlap with all existing issues and PRs
    for existing in intel.get("issues", []) + intel.get("prs", []):
        ext_title = existing.get("title", "").strip()
        num = existing.get("number", "?")
        state = existing.get("state", "UNKNOWN")
        ext_tokens = extract_keywords(ext_title)
        if not ext_tokens:
            continue

        overlap = cand_tokens & ext_tokens
        similarity = len(overlap) / len(cand_tokens | ext_tokens)

        # High Jaccard similarity or core technical token match
        if similarity >= 0.40 or (len(overlap) >= 3 and len(overlap) / len(ext_tokens) >= 0.60):
            return True, f"High semantic overlap ({similarity:.2f}) with existing issue/PR #{num} (State: {state}): '{ext_title}' (Shared terms: {list(overlap)})"

    # Check 3: Codebase file presence check
    # If the target file already exists in the repo and has substantial content, the feature may already be implemented!
    for tf in target_files:
        norm_tf = tf.replace("\\", "/")
        if norm_tf in intel.get("code_files", set()):
            target_path = workspace / norm_tf
            if target_path.is_file():
                try:
                    content = target_path.read_text(encoding="utf-8", errors="replace")
                    # If the file exists and is > 40 lines, check if candidate's core keywords appear in it
                    if len(content.splitlines()) > 40:
                        file_tokens = extract_keywords(content)
                        matches = cand_tokens & file_tokens
                        if len(matches) >= 3:
                            return True, f"Target file '{norm_tf}' already exists in codebase and implements core functionality (Matched tokens: {list(matches)})"
                except Exception:
                    pass

    # Check 4: Deep LLM Deduplication Gate (Gemini)
    api_key = os.environ.get("GEMINI_REVIEWER_KEY") or os.environ.get("GEMINI_ISSUE_KEY") or os.environ.get("GEMINI_API_KEY")
    if api_key:
        # Build compact digest of recent issues & PRs
        digest_lines = []
        for i in intel.get("issues", [])[:40]:
            digest_lines.append(f"- Issue #{i.get('number')}: [{i.get('state')}] {i.get('title')}")
        for p in intel.get("prs", [])[:20]:
            digest_lines.append(f"- PR #{p.get('number')}: [{p.get('state')}] {p.get('title')}")
        digest_str = "\n".join(digest_lines)

        dedup_prompt = f"""You are the Lead Systems Architect auditing a proposed new GitHub issue for DevProxy (Go high-throughput reverse proxy).
Your mandate is to prevent duplicate issues. Determine if this proposed issue is:
1. ALREADY REPORTED in any open issue or PR.
2. ALREADY SOLVED in any closed issue, merged PR, or existing codebase feature.
3. SUBSTANTIALLY REDUNDANT with an existing capability.

PROPOSED ISSUE:
Title: {title}
Domain: {item.get('area')}
Target Files: {target_files}
Specification: {spec}

EXISTING REPOSITORY ISSUES & PULL REQUESTS:
{digest_str}

Respond ONLY with JSON matching this schema:
{{
  "is_duplicate_or_solved": true or false,
  "confidence": 0 to 100,
  "conflicting_ref": "Issue #X or PR #Y if duplicate, or null",
  "reason": "Clear explanation of why it is duplicate/already solved, or why it is novel"
}}
"""
        for model in FALLBACK_MODELS:
            try:
                url = f"https://generativelanguage.googleapis.com/v1beta/models/{model}:generateContent?key={api_key}"
                payload = json.dumps({
                    "contents": [{"parts": [{"text": dedup_prompt}]}],
                    "generationConfig": {"temperature": 0.1, "responseMimeType": "application/json"}
                }).encode("utf-8")
                req = urllib.request.Request(url, data=payload, headers={"Content-Type": "application/json"}, method="POST")
                with urllib.request.urlopen(req, timeout=15) as resp:
                    res_data = json.loads(resp.read().decode("utf-8"))
                    res_text = res_data["candidates"][0]["content"]["parts"][0]["text"]
                    dedup_res = json.loads(res_text)
                    if dedup_res.get("is_duplicate_or_solved"):
                        conf = dedup_res.get("conflicting_ref") or "existing task"
                        reason = dedup_res.get("reason", "Detected as duplicate by LLM audit")
                        return True, f"LLM Deduplication Gate flagged duplicate of {conf}: {reason}"
                    break
            except Exception:
                continue

    return False, "Verified novel and unaddressed."

def evaluate_issue_quality(item: dict) -> tuple[bool, int, str]:
    """
    Strict Pre-Flight Quality Gate:
    Ensures that ONLY top-tier, production-grade architectural advancements are created.
    Rejects trivial tasks, vague specs, and superficial chores.
    """
    title = item.get("title", "").strip()
    spec = item.get("spec", "").strip()
    target_files = item.get("target_files", [])

    # 1. Title formatting check
    if not any(title.startswith(p) for p in ("feat(", "perf(", "fix(", "refactor(", "security(")):
        return False, 30, "Title must follow Conventional Commits (e.g. feat(proxy): ..., perf(metrics): ...)"

    # 2. Strict Anti-Triviality Blacklist
    trivial_keywords = [
        "readme", "documentation", "typo", "comment", "lint", "variable name",
        "format code", "bump dependency", "clean up", "cosmetic", "rename", "minor",
        "trivial", "update text", "fix grammar"
    ]
    for kw in trivial_keywords:
        if kw in title.lower() or kw in spec.lower():
            return False, 20, f"Task contains trivial keyword '{kw}'. Only deep systems engineering tasks are permitted."

    # 3. Systems Engineering Scope Check
    systems_domains = [
        "zero-allocation", "lock-free", "sync.pool", "atomic", "ringbuffer", "latency",
        "throughput", "streaming", "concurrency", "race", "goroutine", "protocol",
        "http/2", "http/3", "quic", "grpc", "protobuf", "tls", "ja4", "fingerprint",
        "aho-corasick", "simd", "smuggling", "proxy", "waf", "backpressure", "har",
        "opentelemetry", "trace", "connection pool", "keepalive", "benchmark", "wasm",
        "dns", "graphql", "zstd", "ratelimit"
    ]
    matched_domains = [kw for kw in systems_domains if kw in spec.lower() or kw in title.lower()]
    if len(matched_domains) < 2:
        return False, 45, f"Specification lacks technical depth. Must touch core systems domains (matched only {matched_domains})."

    # 4. Strict Specification Rigor
    if len(spec) < 100:
        return False, 50, "Specification is too brief. Must detail architecture, performance constraints, and invariants."

    if not target_files:
        return False, 60, "Must specify target files in pkg/ to ensure clear module decoupling."

    return True, 95, "Approved via strict deterministic systems heuristics."

def check_freeze_timer(existing_issues, cooldown_minutes=30, force=False):
    """Checks if an advancement issue was submitted recently."""
    if force:
        return True, 999.0

    now = datetime.now(timezone.utc)
    for issue in existing_issues:
        created_at_str = issue.get("createdAt")
        if not created_at_str:
            continue
        try:
            dt = datetime.fromisoformat(created_at_str.replace("Z", "+00:00"))
            elapsed = (now - dt).total_seconds() / 60.0
            if elapsed < cooldown_minutes:
                return False, elapsed
        except Exception:
            pass
        break

    return True, 999.0

def load_backlog(workspace: Path) -> list:
    backlog_file = workspace / ".github" / "advancement_backlog.json"
    if backlog_file.is_file():
        try:
            return json.loads(backlog_file.read_text(encoding="utf-8"))
        except Exception:
            return []
    return []

def save_backlog(workspace: Path, backlog: list):
    backlog_file = workspace / ".github" / "advancement_backlog.json"
    backlog_file.parent.mkdir(parents=True, exist_ok=True)
    backlog_file.write_text(json.dumps(backlog, indent=2), encoding="utf-8")

def generate_ai_advancement(intel: dict) -> dict:
    """Uses Gemini API to dynamically generate a novel, high-impact architectural advancement task."""
    api_key = os.environ.get("GEMINI_ISSUE_KEY") or os.environ.get("GEMINI_API_KEY")
    if not api_key:
        return None

    memes_context = ""
    if GOSSIP_AVAILABLE:
        try:
            kb = MemeticKnowledgeBase()
            memes_context = kb.format_prompt_context(repo="DevProxy")
        except Exception:
            pass

    existing_titles_list = list(intel.get("all_titles", []))[:50]

    prompt = f"""You are the Principal Systems Software Architect of DevProxy, an ultra-high performance HTTP/HTTPS reverse proxy and network analysis engine written in Go.

Create a brand new, novel, high-impact systems engineering task for open-source contributors.

STRICT HIGH-QUALITY ARCHITECTURAL INVARIANTS:
1. NO TRIVIAL CHORES: NEVER generate documentation fixes, typo corrections, dependency bumps, cosmetic UI adjustments, or trivial variable renames.
2. PRODUCTION SYSTEMS FOCUS: Tasks must solve hard problems: network protocol parsing, zero-allocation buffer pooling (sync.Pool), lock-free atomic concurrency, WAF security filters, distributed telemetry, or TLS cryptography.
3. ANTI-SPAGHETTI DESIGN: Every feature must specify clean package decoupling, single-responsibility functions (<60 LOC), and zero circular dependencies.

{memes_context if memes_context else ""}

CRITICAL DEDUPLICATION REQUIREMENT:
DO NOT duplicate, overlap, or solve any feature mentioned in these existing titles:
{json.dumps(existing_titles_list, indent=2)}

Output ONLY valid JSON matching this schema:
{{
  "title": "feat(subsystem): Concise descriptive title",
  "area": "area/proxy" or "area/security" or "area/replay" or "area/dashboard" or "area/metrics",
  "difficulty": "enhancement" or "priority/high",
  "spec": "Clear 3-4 sentence technical specification outlining what to implement, performance invariants, zero-allocation memory constraints, and target packages.",
  "target_files": ["pkg/proxy/...", "pkg/analysis/..."]
}}
"""
    payload = {
        "contents": [{"parts": [{"text": prompt}]}],
        "generationConfig": {"temperature": 0.3, "responseMimeType": "application/json"}
    }

    for model in FALLBACK_MODELS:
        url = f"https://generativelanguage.googleapis.com/v1beta/models/{model}:generateContent?key={api_key}"
        for attempt in range(1, 4):
            try:
                req = urllib.request.Request(
                    url,
                    data=json.dumps(payload).encode("utf-8"),
                    headers={"Content-Type": "application/json"},
                    method="POST"
                )
                with urllib.request.urlopen(req, timeout=25) as resp:
                    data = json.loads(resp.read().decode("utf-8"))
                    text = data["candidates"][0]["content"]["parts"][0]["text"]
                    candidate = json.loads(text)
                    return candidate
            except urllib.error.HTTPError as e:
                if e.code == 429:
                    freeze_sec = 15 * attempt
                    print(f"[FREEZE TIMER] Rate limit encountered on {model}. Backing off for {freeze_sec}s...", flush=True)
                    time.sleep(freeze_sec)
                    continue
                break
            except Exception as e:
                print(f"[WARN] Error with {model}: {e}")
                break
    return None

def create_issue(item: dict, intel: dict, workspace: Path) -> bool:
    """Evaluates quality gate, deduplication gate, and creates a structured GitHub issue."""
    title = item.get("title", "").strip()

    # 1. Deduplication Gate: Check if already reported or solved
    is_dup, dup_reason = is_already_reported_or_solved(item, intel, workspace)
    if is_dup:
        print(f"[DEDUPLICATION REJECTED] '{title}' was rejected because it is already reported or solved: {dup_reason}", flush=True)
        return False

    # 2. Enforce Pre-Flight Quality Gate
    is_approved, score, quality_reason = evaluate_issue_quality(item)
    if not is_approved or score < 90:
        print(f"[QUALITY GATE REJECTED] '{title}' failed quality standards (Score: {score}/100): {quality_reason}", flush=True)
        return False

    area = item.get("area", "area/proxy")
    difficulty = item.get("difficulty", "enhancement")
    spec = item.get("spec", "")
    target_files = item.get("target_files", [])

    target_files_md = "\n".join(f"- `{f}`" for f in target_files) if target_files else "- Relevant files in `pkg/`"

    body = f"""## 🚀 Architecture Advancement Specification

### 📌 Architectural Rationale & Threat/Performance Model
{spec}

### 🎯 Subsystem & Domain
- **Domain**: `{area}`
- **Difficulty**: `{difficulty}`
- **Quality Verification**: `Verified Architectural Gate (Quality Score: {score}/100)`
- **Deduplication Check**: `Verified Novel and Unaddressed`
- **Initiative**: Hacktoberfest / Sovereign High-Performance DevProxy Advancement

### 📂 Target Files & Modules
{target_files_md}

### 📋 Technical Acceptance Criteria & Quantitative Invariants
- [ ] Conforms to DevProxy's zero-allocation streaming patterns (utilize `sync.Pool` for buffers).
- [ ] Maintains deterministic performance ($O(1)$ lookup or $O(N)$ streaming throughput without whole-body memory buffering).
- [ ] Concurrency safety verified: zero data races, proper mutex/atomic synchronization, no goroutine leaks on context cancellation.
- [ ] Table-driven Go unit tests added covering normal operation, boundary conditions, and network error branches.
- [ ] Code formatted with `gofmt` and static analysis clean (`go vet ./...` & `go test -race ./...`).

---

### 🍝 Anti-Spaghetti Code Warning & Architecture Invariants
> ⚠️ **STRICT CODE REVIEWER STANDARDS**: Any implementation that introduces spaghetti code will be automatically rejected by the Autonomous Reviewer Bot!
> 
> - **Modular Architecture**: Keep functions concise (<60 LOC), single-purpose, and decoupled across `pkg/proxy`, `pkg/analysis`, `pkg/ringbuffer`, `pkg/dashboard`.
> - **Zero Data Races**: Enforce thread safety using `sync.RWMutex`, `sync.Once`, atomic values, or Go channels. Goroutines must terminate cleanly on `ctx.Done()`.
> - **Zero-Allocation Hot Paths**: Utilize `sync.Pool` for byte buffers (`[]byte`). Avoid heap allocations on proxy request forwarding loops.
> - **Streaming Invariants**: Stream payloads via `io.Reader`/`io.Writer` rather than buffering whole multi-megabyte payloads in memory.
> - **Table-Driven Tests**: Provide comprehensive Go unit tests covering happy paths, edge cases, and error branches.

### 🛠️ Verification Commands
```bash
go mod tidy
gofmt -w .
go vet ./...
go test -race -v ./pkg/...
```
"""

    labels = f"{area},{difficulty},hacktoberfest,advancement"
    cmd = [
        "gh", "issue", "create",
        "--title", title,
        "--body", body,
        "--label", labels
    ]
    try:
        result = subprocess.run(cmd, cwd=workspace, capture_output=True, text=True, check=True)
        created_url = result.stdout.strip()
        print(f"[SUCCESS] Created High-Quality Issue (Score: {score}/100): {created_url} - {title}", flush=True)

        # Update intel
        intel["all_titles"].add(title)
        intel["open_issues"].append({"title": title, "state": "OPEN"})

        # Broadcast newly approved architectural challenge to GossipMesh
        if GOSSIP_AVAILABLE:
            try:
                node = GossipNode("devproxy_issue_generator")
                node.publish(
                    GossipTopic.ARCHITECTURAL_RULES,
                    {"title": title, "spec": spec, "score": score},
                    repo="DevProxy"
                )
            except Exception:
                pass

        return True
    except subprocess.CalledProcessError as e:
        print(f"[ERROR] Failed to create issue '{title}': {e.stderr}", flush=True)
        return False

def main():
    workspace = Path(".").resolve()
    count = int(os.environ.get("INPUT_COUNT", "1") or "1")
    cooldown_min = int(os.environ.get("MIN_COOLDOWN_MINUTES", "30"))
    force = os.environ.get("FORCE_SUBMIT", "false").lower() in ("true", "1") or ("--force" in sys.argv) or ("-f" in sys.argv)

    print("[*] Gathering repository intelligence (issues, PRs, codebase files)...", flush=True)
    intel = get_repository_intel(workspace)
    print(f"[*] Found {len(intel['issues'])} issues, {len(intel['prs'])} PRs, and {len(intel['code_files'])} Go source files in DevProxy.", flush=True)

    # 1. Check Freeze Timer Cooldown
    can_submit, elapsed = check_freeze_timer(intel["issues"], cooldown_minutes=cooldown_min, force=force)
    if not can_submit:
        print(f"[FREEZE TIMER ACTIVE] Last issue was created {elapsed:.1f} minutes ago (< {cooldown_min} min cooldown).", flush=True)
        print("[*] Generating next high-quality advancement and archiving to backlog queue...", flush=True)
        candidate = None
        for item in CATALOG:
            is_dup, _ = is_already_reported_or_solved(item, intel, workspace)
            if not is_dup:
                is_app, score, _ = evaluate_issue_quality(item)
                if is_app and score >= 90:
                    candidate = item
                    break
        if not candidate:
            for _ in range(3):
                ai_cand = generate_ai_advancement(intel)
                if ai_cand:
                    is_dup, _ = is_already_reported_or_solved(ai_cand, intel, workspace)
                    if not is_dup:
                        is_app, score, _ = evaluate_issue_quality(ai_cand)
                        if is_app and score >= 90:
                            candidate = ai_cand
                            break

        if candidate:
            backlog = load_backlog(workspace)
            if not any(b["title"] == candidate["title"] for b in backlog):
                backlog.append(candidate)
                save_backlog(workspace, backlog)
                print(f"[ARCHIVED TO BACKLOG] Stored novel issue: '{candidate['title']}'.", flush=True)
        return

    # 2. Cooldown is clear: Drain backlog first if available
    backlog = load_backlog(workspace)
    created_count = 0

    while backlog and created_count < count:
        item = backlog.pop(0)
        is_dup, _ = is_already_reported_or_solved(item, intel, workspace)
        if not is_dup:
            print(f"[*] Submitting archived backlog item: '{item['title']}'...", flush=True)
            if create_issue(item, intel, workspace):
                created_count += 1
                save_backlog(workspace, backlog)
                if created_count < count:
                    time.sleep(12)

    # 3. Pull from curated high-quality catalog
    for item in CATALOG:
        if created_count >= count:
            break
        is_dup, _ = is_already_reported_or_solved(item, intel, workspace)
        if not is_dup:
            if create_issue(item, intel, workspace):
                created_count += 1
                if created_count < count:
                    time.sleep(12)

    # 4. If catalog exhausted and more requested, dynamically generate via Gemini AI with quality & deduplication gates
    if created_count < count:
        print(f"[*] Catalog exhausted or more issues requested ({created_count}/{count}). Querying Gemini AI with strict deduplication gate...", flush=True)
        max_attempts = 5
        attempts = 0
        while created_count < count and attempts < max_attempts:
            attempts += 1
            ai_item = generate_ai_advancement(intel)
            if not ai_item:
                continue
            is_dup, reason = is_already_reported_or_solved(ai_item, intel, workspace)
            if is_dup:
                print(f"[*] AI candidate '{ai_item.get('title')}' rejected: {reason}", flush=True)
                continue
            if create_issue(ai_item, intel, workspace):
                created_count += 1
                if created_count < count:
                    time.sleep(12)

    if created_count == 0:
        print("[OK] No new issues needed or all candidate specifications are already reported/solved.")
    else:
        print(f"[DONE] Successfully created {created_count} verified high-quality advancement issue(s).")

if __name__ == "__main__":
    main()
