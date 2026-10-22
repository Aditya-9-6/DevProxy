$org = Invoke-RestMethod -Method Post -Uri http://127.0.0.1:8001/admin/orgs -ContentType 'application/json' -Body '{"name":"Acme Corp"}'
$token = $org.token
Write-Host "TOKEN=$token"

# Heartbeat
Invoke-RestMethod -Method Post -Uri http://127.0.0.1:8001/v1/nodes/heartbeat -Headers @{Authorization = "Bearer $token"; "Content-Type" = "application/json"} -Body '{"node_name":"production-proxy-01","version":"0.1.0"}'

# Telemetry ingest
Invoke-RestMethod -Method Post -Uri http://127.0.0.1:8001/v1/telemetry/ingest -Headers @{Authorization = "Bearer $token"; "Content-Type" = "application/json"} -Body '{"method":"POST","path":"/api/v1/checkout","status_code":500,"error_snippet":"TypeError: NoneType has no attribute checkout","redacted_headers":{},"latency_ms":145}'

# Metrics
Invoke-RestMethod -Uri http://127.0.0.1:8001/metrics/metrics
