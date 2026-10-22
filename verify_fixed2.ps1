$orgResp = Invoke-RestMethod -Method POST -Uri "http://127.0.0.1:8002/admin/orgs?name=Acme%20Corp"
$token = $orgResp.token
Write-Host "TOKEN=$token"

# Heartbeat
$hbBody = @{node_name="production-proxy-01"; version="0.1.0"} | ConvertTo-Json -Compress
Invoke-RestMethod -Method POST -Uri "http://127.0.0.1:8002/v1/nodes/heartbeat" -Headers @{Authorization="Bearer $token"; "Content-Type"="application/json"} -Body $hbBody

# Telemetry ingest
$teleBody = @{method="POST"; path="/api/v1/checkout"; status_code=500; redacted_headers=@{}; redacted_request_body=""; redacted_response_body=""; latency_ms=145} | ConvertTo-Json -Compress
Invoke-RestMethod -Method POST -Uri "http://127.0.0.1:8002/v1/telemetry/ingest" -Headers @{Authorization="Bearer $token"; "Content-Type"="application/json"} -Body $teleBody

# Metrics
Invoke-RestMethod -Method GET -Uri "http://127.0.0.1:8002/metrics/metrics"
