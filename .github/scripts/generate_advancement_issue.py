#!/usr/bin/env python3
"""
DevProxy Architecture Advancement Issue Generator Engine
Autonomous issue generator bot for continuous open-source contribution & architectural advancement.
Features:
- Curated high-impact systems engineering catalog
- Dynamic Gemini AI generation with strict anti-spaghetti prompts
- Freeze timer & cooldown protection to prevent GitHub secondary rate-limit breaks
- Local backlog queue archiving when cooldown is active
- Mandatory anti-spaghetti warnings and architectural quality criteria
"""

import json
import os
import subprocess
import sys
import time
import urllib.request
import urllib.error
from datetime import datetime, timezone
from pathlib import Path

DEFAULT_MODEL = "gemini-2.5-flash"
FALLBACK_MODELS = ["gemini-2.5-flash", "gemini-2.5-pro", "gemini-2.0-flash"]

# Curated catalog of high-impact open-source architecture advancements for DevProxy (Go)
CATALOG = [
    {
        "title": "feat(tls): Add JA4+ TLS Client Fingerprinting & Bot Classifier",
        "area": "area/security",
        "difficulty": "enhancement",
        "spec": "Capture TLS ClientHello extension lists, cipher suites, ALPN protocols, and signature algorithms to generate standard JA4 and JA4S fingerprints. Implement an in-memory classifier to flag suspicious client fingerprints diverging from claimed User-Agent headers.",
        "target_files": ["pkg/proxy/proxy.go", "pkg/analysis/rules.go", "pkg/ringbuffer/event.go"]
    },
    {
        "title": "feat(proxy): Add gRPC & Protobuf Binary Stream Decoder in Dashboard",
        "area": "area/proxy",
        "difficulty": "enhancement",
        "spec": "Detect application/grpc and application/grpc+proto streams over HTTP/2. Parse standard 5-byte gRPC framing headers (compressed flag + 4-byte big-endian length prefix) and decode structured protobuf fields to display readable JSON in the Web Dashboard.",
        "target_files": ["pkg/proxy/grpc.go", "pkg/dashboard/hub.go", "web/index.html"]
    },
    {
        "title": "feat(proxy): Implement Upstream Dynamic Proxy Chaining (SOCKS5 & HTTP Connect)",
        "area": "area/proxy",
        "difficulty": "good first issue",
        "spec": "Add support for an -upstream-proxy CLI flag supporting socks5:// and http:// corporate egress proxies. Configure http.Transport.Proxy dialer to transparently tunnel outbound proxy requests through the corporate boundary.",
        "target_files": ["pkg/proxy/proxy.go", "cmd/devproxy/main.go"]
    },
    {
        "title": "feat(tracing): Add Distributed OpenTelemetry (OTel) W3C Context Propagation",
        "area": "area/proxy",
        "difficulty": "good first issue",
        "spec": "Extract incoming W3C traceparent and tracestate headers and inject them into downstream proxy requests. Record trace IDs in TrafficEvent to enable end-to-end distributed trace tracking across microservices.",
        "target_files": ["pkg/proxy/proxy.go", "pkg/ringbuffer/event.go", "pkg/dashboard/hub.go"]
    },
    {
        "title": "feat(replay): Implement Deterministic HAR (HTTP Archive) Recording and Playback",
        "area": "area/replay",
        "difficulty": "good first issue",
        "spec": "Export captured ringbuffer traffic to valid HAR 1.2 format JSON files. Support a devproxy replay -har=session.har command that mocks endpoints according to previously recorded session timings and status codes.",
        "target_files": ["pkg/replay/har.go", "pkg/mock/server.go", "cmd/devproxy/main.go"]
    },
    {
        "title": "feat(security): Add SIMD-Accelerated Aho-Corasick Multi-Pattern Secret Scanner",
        "area": "area/security",
        "difficulty": "enhancement",
        "spec": "Deploy pre-compiled Aho-Corasick automaton for simultaneous matching of API keys (AWS, Stripe, GitHub, OpenAI) in HTTP request bodies with sub-microsecond inspection latency.",
        "target_files": ["pkg/analysis/rules.go", "pkg/analysis/finding.go"]
    },
    {
        "title": "feat(metrics): Add Lock-Free HDR Latency Histograms for P99/P99.9 Percentiles",
        "area": "area/proxy",
        "difficulty": "good first issue",
        "spec": "Implement atomic HDR histogram collection tracking P50, P90, P99, and P99.9 latency percentiles across proxy endpoints. Expose metrics at /metrics Prometheus endpoint.",
        "target_files": ["pkg/proxy/metrics.go", "pkg/dashboard/server.go"]
    },
    {
        "title": "feat(proxy): Add Dynamic Upstream Connection Pooling with Active Keepalive Probes",
        "area": "area/proxy",
        "difficulty": "enhancement",
        "spec": "Implement active health-check probes on idle HTTP/1.1 and HTTP/2 upstream TCP connections in http.Transport to eliminate 502 Bad Gateway race conditions caused by silently closed server sockets.",
        "target_files": ["pkg/proxy/proxy.go", "pkg/proxy/upstream.go"]
    },
    {
        "title": "feat(waf): Add HTTP Request Smuggling (CL.TE / TE.CL) Desynchronization Detector",
        "area": "area/security",
        "difficulty": "enhancement",
        "spec": "Analyze incoming request headers for conflicting Content-Length and Transfer-Encoding headers, whitespace obfuscation, and duplicate headers to detect desynchronization smuggling exploits before dispatch.",
        "target_files": ["pkg/analysis/rules.go", "pkg/proxy/proxy.go"]
    },
    {
        "title": "feat(dashboard): Add WebSocket Backpressure & Zero-Copy Packet Telemetry Streamer",
        "area": "area/dashboard",
        "difficulty": "good first issue",
        "spec": "Implement bounded ring-buffer dispatch for the WebSocket hub to drop outdated telemetry frames under slow client network conditions, preventing proxy worker memory ballooning.",
        "target_files": ["pkg/dashboard/hub.go", "pkg/dashboard/server.go"]
    }
]

def get_existing_issues():
    """Fetch existing issue titles and creation timestamps."""
    issues_info = []
    try:
        cmd = ["gh", "issue", "list", "--state", "all", "--limit", "100", "--json", "title,createdAt,author"]
        result = subprocess.run(cmd, capture_output=True, text=True, check=True)
        issues = json.loads(result.stdout)
        return issues
    except Exception as e:
        print(f"[WARN] Failed to fetch issues via gh CLI: {e}")
        return []

def check_freeze_timer(existing_issues, cooldown_minutes=30, force=False):
    """
    Checks if an advancement issue was submitted recently.
    Returns (can_submit: bool, minutes_elapsed: float)
    """
    if force:
        return True, 999.0

    now = datetime.now(timezone.utc)
    for issue in existing_issues:
        created_at_str = issue.get("createdAt")
        if not created_at_str:
            continue
        try:
            # Parse ISO 8601
            dt = datetime.fromisoformat(created_at_str.replace("Z", "+00:00"))
            elapsed = (now - dt).total_seconds() / 60.0
            if elapsed < cooldown_minutes:
                return False, elapsed
        except Exception:
            pass
        # Only inspect the most recent issue
        break

    return True, 999.0

def load_backlog(workspace: Path) -> list:
    """Loads archived backlog queue from file."""
    backlog_file = workspace / ".github" / "advancement_backlog.json"
    if backlog_file.is_file():
        try:
            return json.loads(backlog_file.read_text(encoding="utf-8"))
        except Exception:
            return []
    return []

def save_backlog(workspace: Path, backlog: list):
    """Saves backlog queue to file."""
    backlog_file = workspace / ".github" / "advancement_backlog.json"
    backlog_file.parent.mkdir(parents=True, exist_ok=True)
    backlog_file.write_text(json.dumps(backlog, indent=2), encoding="utf-8")

def generate_ai_advancement(existing_titles):
    """Use Gemini API to dynamically generate a novel, high-impact architectural advancement task."""
    api_key = os.environ.get("GEMINI_ISSUE_KEY") or os.environ.get("GEMINI_API_KEY")
    if not api_key:
        return None

    prompt = f"""You are the Principal Systems Software Architect of DevProxy, an ultra-high performance HTTP/HTTPS reverse proxy and network analysis engine written in Go.

Create a brand new, novel, high-impact systems engineering task for open-source contributors.

STRICT QUALITY INVARIANTS:
1. NO TRIVIAL CHORES: Do NOT generate documentation fixes, typo corrections, dependency bumps, cosmetic UI adjustments, or trivial variable renames.
2. PRODUCTION SYSTEMS FOCUS: Tasks must involve network protocols, high-throughput streaming, zero-allocation buffer pooling (sync.Pool), concurrency safety, security/WAF detection, distributed telemetry, or TLS cryptography.
3. ANTI-SPAGHETTI DESIGN: Every feature must specify clean package decoupling, single-responsibility functions (<60 LOC), and zero circular dependencies.

DO NOT duplicate any of these existing titles:
{json.dumps(list(existing_titles)[:30], indent=2)}

Output ONLY valid JSON matching this schema:
{{
  "title": "feat(subsystem): Concise descriptive title",
  "area": "area/proxy" or "area/security" or "area/replay" or "area/dashboard" or "area/metrics",
  "difficulty": "good first issue" or "enhancement" or "priority/high",
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
                    return json.loads(text)
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

def create_issue(item):
    """Create a structured GitHub issue with strict anti-spaghetti warnings."""
    title = item["title"]
    area = item.get("area", "area/proxy")
    difficulty = item.get("difficulty", "good first issue")
    spec = item["spec"]
    target_files = item.get("target_files", [])

    target_files_md = "\n".join(f"- `{f}`" for f in target_files) if target_files else "- Relevant files in `pkg/`"

    body = f"""## 🚀 Architecture Advancement Specification

### 📌 Overview & Rationale
{spec}

### 🎯 Subsystem & Domain
- **Domain**: `{area}`
- **Difficulty**: `{difficulty}`
- **Initiative**: Hacktoberfest / Sovereign High-Performance DevProxy Advancement

### 📂 Target Files & Modules
{target_files_md}

### 📋 Technical Acceptance Criteria
- [ ] Conforms to DevProxy's zero-allocation streaming patterns (utilize `sync.Pool` for buffers).
- [ ] Maintains deterministic performance ($O(1)$ lookup or $O(N)$ streaming throughput).
- [ ] Concurrency safety verified: zero data races, proper mutex/atomic synchronization, no goroutine leaks on context cancellation.
- [ ] Unit tests added covering normal operation, boundary conditions, and error branches.
- [ ] Code formatted with `gofmt` and static checks clean (`go vet ./...`).

---

### 🍝 Anti-Spaghetti Code Warning & Architecture Invariants
> ⚠️ **STRICT CODE REVIEWER STANDARDS**: Any implementation that introduces spaghetti code will be automatically rejected by the Autonomous Reviewer Bot!
> 
> - **Modular Architecture**: Keep functions concise (<60 LOC), single-purpose, and decoupled across `pkg/proxy`, `pkg/analysis`, `pkg/ringbuffer`, `pkg/dashboard`.
> - **Zero Data Races**: Enforce thread safety using `sync.RWMutex`, `sync.Once`, atomic values, or Go channels. Goroutines must terminate cleanly on `ctx.Done()`.
> - **Zero-Allocation Hot Paths**: Utilize `sync.Pool` for byte buffers (`[]byte`). Avoid heap allocations on proxy request forwarding loops.
> - **Streaming Invariants**: Stream payloads via `io.Reader`/`io.Writer` rather than buffering whole multi-megabyte payloads in memory.
> - **Table-Driven Tests**: Provide comprehensive Go unit tests covering happy paths, edge cases, and error branches.

### 🛠️ Getting Started
1. Fork the repository and create a branch: `git checkout -b feature/{area.replace('area/', '')}-advancement`
2. Run local tests: `go test -v ./...`
3. When ready, open a PR. Comment `@devproxy-bot solve` or `/solve` to ask the AI assistant for scaffolding, or submit your solution for automated review!
"""

    labels = f"{area},{difficulty},hacktoberfest,advancement"
    cmd = [
        "gh", "issue", "create",
        "--title", title,
        "--body", body,
        "--label", labels
    ]
    try:
        result = subprocess.run(cmd, capture_output=True, text=True, check=True)
        print(f"[SUCCESS] Created issue: {result.stdout.strip()} - {title}")
        return True
    except subprocess.CalledProcessError as e:
        print(f"[ERROR] Failed to create issue '{title}': {e.stderr}")
        return False

def main():
    workspace = Path(".").resolve()
    manual_title = os.environ.get("INPUT_TITLE", "").strip()
    manual_area = os.environ.get("INPUT_AREA", "auto").strip()
    manual_diff = os.environ.get("INPUT_DIFFICULTY", "good first issue").strip()
    manual_spec = os.environ.get("INPUT_SPEC", "").strip()
    count = int(os.environ.get("INPUT_COUNT", "1") or "1")
    cooldown_min = int(os.environ.get("MIN_COOLDOWN_MINUTES", "30"))
    force = os.environ.get("FORCE_SUBMIT", "false").lower() in ("true", "1") or ("--force" in sys.argv) or ("-f" in sys.argv)

    # If manual submission specified
    if manual_title and manual_area != "auto":
        create_issue({
            "title": manual_title,
            "area": manual_area,
            "difficulty": manual_diff,
            "spec": manual_spec or f"Implement architectural advancement for {manual_title}."
        })
        return

    existing_issues = get_existing_issues()
    existing_titles = {i["title"].strip() for i in existing_issues if "title" in i}
    print(f"[*] Found {len(existing_titles)} existing issues in repository.")

    # 1. Check Freeze Timer Cooldown
    can_submit, elapsed = check_freeze_timer(existing_issues, cooldown_minutes=cooldown_min, force=force)
    if not can_submit:
        print(f"[FREEZE TIMER ACTIVE] Last issue was created {elapsed:.1f} minutes ago (< {cooldown_min} min cooldown).")
        print("[*] Generating next high-impact advancement and archiving to backlog queue to prevent rate-limit breaks...")

        # Find next candidate to archive
        candidate = None
        for item in CATALOG:
            if item["title"] not in existing_titles:
                candidate = item
                break
        if not candidate:
            candidate = generate_ai_advancement(existing_titles)

        if candidate:
            backlog = load_backlog(workspace)
            if not any(b["title"] == candidate["title"] for b in backlog):
                backlog.append(candidate)
                save_backlog(workspace, backlog)
                print(f"[ARCHIVED TO BACKLOG] Stored: '{candidate['title']}'. Will be submitted once freeze timer expires.")
        return

    # 2. Cooldown is clear: Drain backlog first if available
    backlog = load_backlog(workspace)
    created_count = 0

    while backlog and created_count < count:
        item = backlog.pop(0)
        if item["title"] not in existing_titles:
            print(f"[*] Submitting archived backlog item: '{item['title']}'...")
            if create_issue(item):
                existing_titles.add(item["title"])
                created_count += 1
                save_backlog(workspace, backlog)
                if created_count < count:
                    print("[RATE-LIMIT DELAY] Sleeping 12s to protect GitHub secondary rate limit...")
                    time.sleep(12)

    # 3. Pull from curated catalog
    for item in CATALOG:
        if created_count >= count:
            break
        if item["title"] not in existing_titles:
            if create_issue(item):
                existing_titles.add(item["title"])
                created_count += 1
                if created_count < count:
                    print("[RATE-LIMIT DELAY] Sleeping 12s to protect GitHub secondary rate limit...")
                    time.sleep(12)

    # 4. If catalog exhausted and more requested, dynamically generate via Gemini AI
    if created_count < count:
        print(f"[*] Catalog exhausted or more issues requested ({created_count}/{count}). Querying Gemini AI for novel advancement...")
        while created_count < count:
            ai_item = generate_ai_advancement(existing_titles)
            if not ai_item or ai_item.get("title") in existing_titles:
                print("[!] Could not generate further unique dynamic issues.")
                break
            if create_issue(ai_item):
                existing_titles.add(ai_item["title"])
                created_count += 1
                if created_count < count:
                    print("[RATE-LIMIT DELAY] Sleeping 12s to protect GitHub secondary rate limit...")
                    time.sleep(12)

    if created_count == 0:
        print("[OK] No new issues needed or all catalog items already exist.")
    else:
        print(f"[DONE] Successfully generated {created_count} advancement issue(s) under safe rate-limiting.")

if __name__ == "__main__":
    main()
