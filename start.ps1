#!/usr/bin/env pwsh
<#
.SYNOPSIS
    Cross-platform startup script for the Risk Platform.

.DESCRIPTION
    Works on Windows, Linux, and macOS (requires Docker, Go 1.24+, Node 20+, Yarn).
    Starts infrastructure via Docker Compose, creates service databases, writes
    .env files (skipped if they already exist), then starts all services and
    the frontend as background jobs.

.PARAMETER InfraOnly
    Stop after starting Docker infrastructure. Do not start Go services or frontend.

.PARAMETER SkipInfra
    Skip Docker Compose startup. Assumes infra is already running.

.PARAMETER SkipEnvSetup
    Skip writing .env files. Use existing files as-is.

.EXAMPLE
    pwsh ./start.ps1               # Full startup
    pwsh ./start.ps1 -InfraOnly    # Only Docker infra
    pwsh ./start.ps1 -SkipInfra    # Only services (infra already up)
#>
param(
    [switch]$InfraOnly,
    [switch]$SkipInfra,
    [switch]$SkipEnvSetup
)

$ErrorActionPreference = "Stop"

$Root        = $PSScriptRoot
$ServicesDir = Join-Path $Root "services"
$InfraDir    = Join-Path $Root "infra" "docker"
$LogsDir     = Join-Path $Root ".logs"

# ─────────────────────────────────────────────────────────────────────────────
# Helpers
# ─────────────────────────────────────────────────────────────────────────────

function Write-Step { param($Msg) Write-Host "`n==> $Msg" -ForegroundColor Cyan }
function Write-OK   { param($Msg) Write-Host "    [ OK ] $Msg" -ForegroundColor Green }
function Write-Warn { param($Msg) Write-Host "    [WARN] $Msg" -ForegroundColor Yellow }
function Write-Fail { param($Msg) Write-Host "    [FAIL] $Msg" -ForegroundColor Red; exit 1 }
function Write-Info { param($Msg) Write-Host "           $Msg" -ForegroundColor Gray }

function Test-Command {
    param($Cmd)
    return [bool](Get-Command $Cmd -ErrorAction SilentlyContinue)
}

function Wait-ForPort {
    param([string]$Host, [int]$Port, [int]$TimeoutSec = 60, [string]$Label = "")
    $deadline = (Get-Date).AddSeconds($TimeoutSec)
    while ((Get-Date) -lt $deadline) {
        try {
            $tcp = [System.Net.Sockets.TcpClient]::new()
            $tcp.Connect($Host, $Port)
            $tcp.Close()
            return $true
        } catch { Start-Sleep -Milliseconds 500 }
    }
    return $false
}

# ─────────────────────────────────────────────────────────────────────────────
# 1. Prerequisites
# ─────────────────────────────────────────────────────────────────────────────

Write-Step "Checking prerequisites"

@("docker", "go") | ForEach-Object {
    if (-not (Test-Command $_)) { Write-Fail "$_ is not installed or not in PATH" }
    Write-OK $_
}

if (-not $InfraOnly) {
    @("node", "yarn") | ForEach-Object {
        if (-not (Test-Command $_)) { Write-Fail "$_ is not installed or not in PATH" }
        Write-OK $_
    }
}

# ─────────────────────────────────────────────────────────────────────────────
# 2. Docker .env
# ─────────────────────────────────────────────────────────────────────────────

$DockerEnv = Join-Path $InfraDir ".env"
if (-not (Test-Path $DockerEnv)) {
    Write-Step "Creating infra/docker/.env"
    @"
KC_BOOTSTRAP_ADMIN_USERNAME=admin
KC_BOOTSTRAP_ADMIN_PASSWORD=admin
KC_DB=postgres
KC_DB_URL=jdbc:postgresql://postgres:5432/keycloak
KC_DB_USERNAME=postgres
KC_DB_PASSWORD=postgres
POSTGRES_USER=postgres
POSTGRES_PASSWORD=postgres
POSTGRES_DB=keycloak
"@ | Set-Content $DockerEnv -Encoding UTF8
    Write-OK "Created $DockerEnv"
} else {
    Write-OK "infra/docker/.env already exists — skipping"
}

# ─────────────────────────────────────────────────────────────────────────────
# 3. Infrastructure (Docker Compose)
# ─────────────────────────────────────────────────────────────────────────────

if (-not $SkipInfra) {
    Write-Step "Starting Docker infrastructure"
    Push-Location $InfraDir
    try {
        docker compose up -d 2>&1 | Out-Null
        Write-OK "docker compose started"
    } catch {
        Write-Fail "docker compose failed: $_"
    } finally {
        Pop-Location
    }

    Write-Step "Waiting for PostgreSQL to be ready (max 60s)"
    if (-not (Wait-ForPort -Host "localhost" -Port 5432 -TimeoutSec 60)) {
        Write-Fail "PostgreSQL did not become available in time"
    }
    Write-OK "PostgreSQL is ready"

    Write-Step "Waiting for Kafka to be ready (max 90s)"
    if (-not (Wait-ForPort -Host "localhost" -Port 9092 -TimeoutSec 90)) {
        Write-Warn "Kafka port not open yet — services may retry on startup"
    } else {
        Write-OK "Kafka is ready"
    }

    Write-Step "Waiting for Keycloak OIDC endpoint to be ready (max 120s)"
    $keycloakReady = $false
    $deadline = (Get-Date).AddSeconds(120)
    while ((Get-Date) -lt $deadline) {
        try {
            $resp = Invoke-WebRequest -Uri "http://localhost:8080/realms/risk-platform-dev/.well-known/openid-configuration" `
                -UseBasicParsing -TimeoutSec 5 -ErrorAction Stop
            if ($resp.StatusCode -eq 200) { $keycloakReady = $true; break }
        } catch { Start-Sleep -Seconds 3 }
    }
    if ($keycloakReady) { Write-OK "Keycloak is ready" }
    else { Write-Warn "Keycloak not ready in time — identity-access will retry on its own" }
} else {
    Write-Warn "-SkipInfra set — skipping Docker Compose startup"

    # Even when skipping infra startup, check Keycloak is reachable before proceeding
    Write-Step "Checking Keycloak OIDC endpoint (max 30s)"
    $keycloakReady = $false
    $deadline = (Get-Date).AddSeconds(30)
    while ((Get-Date) -lt $deadline) {
        try {
            $resp = Invoke-WebRequest -Uri "http://localhost:8080/realms/risk-platform-dev/.well-known/openid-configuration" `
                -UseBasicParsing -TimeoutSec 5 -ErrorAction Stop
            if ($resp.StatusCode -eq 200) { $keycloakReady = $true; break }
        } catch { Start-Sleep -Seconds 3 }
    }
    if ($keycloakReady) { Write-OK "Keycloak is reachable" }
    else { Write-Warn "Keycloak not reachable — identity-access will retry on its own (up to 2 min)" }
}

# ─────────────────────────────────────────────────────────────────────────────
# 4. Create service databases
# ─────────────────────────────────────────────────────────────────────────────

Write-Step "Creating service databases (skip if already exist)"

$databases = @(
    "identity_access", "tenant_config", "ingestion", "rules_engine",
    "decision", "case_management", "workflow", "audit_trail",
    "explanation", "notification", "log_ingestion"
)

foreach ($db in $databases) {
    $out = docker exec postgres psql -U postgres -tc "SELECT 1 FROM pg_database WHERE datname='$db'" 2>&1
    if ($out -match "1") {
        Write-Info "$db already exists"
    } else {
        docker exec postgres psql -U postgres -c "CREATE DATABASE $db;" 2>&1 | Out-Null
        Write-OK "Created database: $db"
    }
}

if ($InfraOnly) {
    Write-Host "`n[Done] Infrastructure is up. Use -SkipInfra to start services next time." -ForegroundColor Cyan
    exit 0
}

# ─────────────────────────────────────────────────────────────────────────────
# 5. Service .env files
# ─────────────────────────────────────────────────────────────────────────────

if (-not $SkipEnvSetup) {
    Write-Step "Writing service .env files (skipped if file already exists)"

    $serviceEnvs = @{
        "identity-access" = @"
GRPC_ADDR=:50051
METRICS_ADDR=:9090
LOG_LEVEL=info
OTEL_SERVICE_NAME=identity-access
OTEL_ENVIRONMENT=local
OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317
KEYCLOAK_BASE_URL=http://localhost:8080
KEYCLOAK_REALM=risk-platform-dev
KEYCLOAK_CLIENT_ID=identity-service-client
KEYCLOAK_CLIENT_SECRET=4bZ1VZcyEwhLCLx3hJ2iu9Tmd84VmKET
KEYCLOAK_ISSUER=http://localhost:8080/realms/risk-platform-dev
KEYCLOAK_AUDIENCE=api-gateway-client
KAFKA_BROKERS=localhost:9092
KAFKA_AUTH_EVENTS_TOPIC=auth.events
"@
        "tenant-config" = @"
GRPC_ADDR=:50052
METRICS_ADDR=:9091
LOG_LEVEL=info
OTEL_SERVICE_NAME=tenant-config
OTEL_ENVIRONMENT=local
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317
POSTGRES_DSN=postgres://postgres:postgres@localhost:5432/tenant_config?sslmode=disable
REDIS_ADDR=localhost:6379
KAFKA_BROKERS=localhost:9092
KAFKA_TENANT_TOPIC=tenant-events
"@
        "ingestion" = @"
GRPC_ADDR=:50053
METRICS_ADDR=:9093
LOG_LEVEL=info
OTEL_SERVICE_NAME=ingestion
OTEL_ENVIRONMENT=local
OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317
POSTGRES_DSN=postgres://postgres:postgres@localhost:5432/ingestion?sslmode=disable
REDIS_ADDR=localhost:6379
KAFKA_BROKERS=localhost:9092
KAFKA_PAYMENTS_TOPIC=payments.received
"@
        "rules-engine" = @"
GRPC_ADDR=:50055
METRICS_ADDR=:9095
LOG_LEVEL=info
OTEL_SERVICE_NAME=rules-engine
OTEL_ENVIRONMENT=local
OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317
POSTGRES_DSN=postgres://postgres:postgres@localhost:5432/rules_engine?sslmode=disable
REDIS_ADDR=localhost:6379
KAFKA_BROKERS=localhost:9092
KAFKA_RULES_TOPIC=rules.evaluated
"@
        "risk-orchestrator" = @"
GRPC_ADDR=:50054
METRICS_ADDR=:9094
LOG_LEVEL=info
OTEL_SERVICE_NAME=risk-orchestrator
OTEL_ENVIRONMENT=local
OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317
REDIS_ADDR=localhost:6379
KAFKA_BROKERS=localhost:9092
KAFKA_PAYMENTS_TOPIC=payments.received
KAFKA_RISK_TOPIC=risk.evaluated
KAFKA_CONSUMER_GROUP=risk-orchestrator
LATENCY_BUDGET_MS=150
RULES_ENGINE_ADDR=localhost:50055
"@
        "decision" = @"
GRPC_ADDR=:50056
METRICS_ADDR=:9096
LOG_LEVEL=info
OTEL_SERVICE_NAME=decision
OTEL_ENVIRONMENT=local
OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317
POSTGRES_DSN=postgres://postgres:postgres@localhost:5432/decision?sslmode=disable
KAFKA_BROKERS=localhost:9092
KAFKA_RISK_TOPIC=risk.evaluated
KAFKA_DECISION_TOPIC=decision.made
KAFKA_CONSUMER_GROUP=decision-service
"@
        "case-management" = @"
GRPC_ADDR=:50057
METRICS_ADDR=:9097
LOG_LEVEL=info
OTEL_SERVICE_NAME=case-management
OTEL_ENVIRONMENT=local
OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317
POSTGRES_DSN=postgres://postgres:postgres@localhost:5432/case_management?sslmode=disable
REDIS_ADDR=localhost:6379
KAFKA_BROKERS=localhost:9092
KAFKA_DECISION_TOPIC=decision.made
KAFKA_CASE_TOPIC=case.created
KAFKA_CONSUMER_GROUP=case-management-service
"@
        "workflow" = @"
GRPC_ADDR=:50058
METRICS_ADDR=:9098
LOG_LEVEL=info
OTEL_SERVICE_NAME=workflow
OTEL_ENVIRONMENT=local
OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317
POSTGRES_DSN=postgres://postgres:postgres@localhost:5432/workflow?sslmode=disable
REDIS_ADDR=localhost:6379
"@
        "audit-trail" = @"
GRPC_ADDR=:50059
METRICS_ADDR=:9099
LOG_LEVEL=info
OTEL_SERVICE_NAME=audit-trail
OTEL_ENVIRONMENT=local
OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317
POSTGRES_DSN=postgres://postgres:postgres@localhost:5432/audit_trail?sslmode=disable
KAFKA_BROKERS=localhost:9092
KAFKA_TOPICS=payments.received,rules.evaluated,risk.evaluated,decision.made,case.created,case.escalated,case.resolved
KAFKA_CONSUMER_GROUP=audit-trail-service
"@
        "explanation" = @"
GRPC_ADDR=:50060
METRICS_ADDR=:9100
LOG_LEVEL=info
OTEL_SERVICE_NAME=explanation
OTEL_ENVIRONMENT=local
OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317
POSTGRES_DSN=postgres://postgres:postgres@localhost:5432/explanation?sslmode=disable
DECISION_SERVICE_ADDR=localhost:50056
RULES_ENGINE_ADDR=localhost:50055
"@
        "notification" = @"
GRPC_ADDR=:50061
METRICS_ADDR=:9101
LOG_LEVEL=info
OTEL_SERVICE_NAME=notification
OTEL_ENVIRONMENT=local
OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317
POSTGRES_DSN=postgres://postgres:postgres@localhost:5432/notification?sslmode=disable
REDIS_ADDR=localhost:6379
KAFKA_BROKERS=localhost:9092
KAFKA_TOPICS=case.created,case.escalated,decision.made
KAFKA_CONSUMER_GROUP=notification-service
"@
        "log-ingestion" = @"
GRPC_ADDR=:50062
METRICS_ADDR=:9102
LOG_LEVEL=info
OTEL_SERVICE_NAME=log-ingestion
OTEL_ENVIRONMENT=local
OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317
POSTGRES_DSN=postgres://postgres:postgres@localhost:5432/log_ingestion?sslmode=disable
KAFKA_BROKERS=localhost:9092
KAFKA_LOGS_TOPIC=ops.logs
KAFKA_CONSUMER_GROUP=log-ingestion-service
"@
        "ops-query" = @"
GRPC_ADDR=:50063
METRICS_ADDR=:9103
LOG_LEVEL=info
OTEL_SERVICE_NAME=ops-query
OTEL_ENVIRONMENT=local
OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317
LOG_INGESTION_ADDR=localhost:50062
PROMETHEUS_ADDR=http://localhost:9090
JAEGER_ADDR=http://localhost:16686
"@
        "graphql-bff" = @"
HTTP_ADDR=:8090
METRICS_ADDR=:9104
LOG_LEVEL=info
SERVICE_NAME=graphql-bff
ENVIRONMENT=development
OTEL_COLLECTOR_ENDPOINT=localhost:4317
REDIS_ADDR=localhost:6379
CORS_ORIGINS=http://localhost:5173
IDENTITY_ACCESS_ADDR=localhost:50051
TENANT_CONFIG_ADDR=localhost:50052
INGESTION_ADDR=localhost:50053
RISK_ORCHESTRATOR_ADDR=localhost:50054
RULES_ENGINE_ADDR=localhost:50055
DECISION_ADDR=localhost:50056
CASE_MANAGEMENT_ADDR=localhost:50057
WORKFLOW_ADDR=localhost:50058
AUDIT_TRAIL_ADDR=localhost:50059
EXPLANATION_ADDR=localhost:50060
NOTIFICATION_ADDR=localhost:50061
LOG_INGESTION_ADDR=localhost:50062
OPS_QUERY_ADDR=localhost:50063
"@
        "frontend" = @"
VITE_GRAPHQL_URL=http://localhost:8090/graphql
VITE_KEYCLOAK_URL=http://localhost:8080
VITE_KEYCLOAK_REALM=risk-platform-dev
VITE_KEYCLOAK_CLIENT_ID=risk-platform-frontend
"@
    }

    foreach ($svc in $serviceEnvs.Keys) {
        $envPath = Join-Path $ServicesDir $svc ".env"
        if (Test-Path $envPath) {
            Write-Info "$svc/.env already exists — skipping"
        } else {
            $serviceEnvs[$svc] | Set-Content $envPath -Encoding UTF8
            Write-OK "Created $svc/.env"
        }
    }
}

# ─────────────────────────────────────────────────────────────────────────────
# 6. Start services as background jobs
# ─────────────────────────────────────────────────────────────────────────────

Write-Step "Creating logs directory"
New-Item -ItemType Directory -Path $LogsDir -Force | Out-Null
Write-OK $LogsDir

# Ordered tiers — ensures dependencies are up before dependents start
$tiers = @(
    @("identity-access", "tenant-config"),               # Tier 1
    @("rules-engine",    "ingestion"),                   # Tier 2
    @("risk-orchestrator","decision"),                   # Tier 3
    @("case-management", "workflow", "explanation",
      "audit-trail",     "log-ingestion"),               # Tier 4
    @("notification",    "ops-query"),                   # Tier 5
    @("graphql-bff")                                     # Tier 6
)

$TierDelaySec = 3   # seconds to wait between tiers
$Jobs = @{}

foreach ($tier in $tiers) {
    foreach ($svc in $tier) {
        $svcDir  = Join-Path $ServicesDir $svc
        $logFile = Join-Path $LogsDir "$svc.log"
        $errFile = Join-Path $LogsDir "$svc.err.log"

        Write-Step "Starting $svc"

        # Check if a job with this name is already running
        $existing = Get-Job -Name $svc -ErrorAction SilentlyContinue
        if ($existing) {
            if ($existing.State -eq "Running") {
                Write-Warn "$svc is already running (job #$($existing.Id)) — skipping"
                $Jobs[$svc] = $existing
                continue
            }
            Remove-Job -Job $existing -Force
        }

        $job = Start-Job -Name $svc -ScriptBlock {
            param($Dir, $LogFile, $ErrFile)
            Set-Location $Dir
            & go run ./cmd/main.go 2>> $ErrFile >> $LogFile
        } -ArgumentList $svcDir, $logFile, $errFile

        $Jobs[$svc] = $job
        Write-OK "$svc started (job #$($job.Id)) — logs: .logs/$svc.log"
    }

    # Brief pause between tiers so earlier services can bind their ports
    if ($tier -ne $tiers[-1]) {
        Write-Info "Waiting ${TierDelaySec}s before next tier..."
        Start-Sleep -Seconds $TierDelaySec
    }
}

# ─────────────────────────────────────────────────────────────────────────────
# 7. Frontend (yarn dev)
# ─────────────────────────────────────────────────────────────────────────────

Write-Step "Starting frontend (yarn dev)"

$frontendDir  = Join-Path $ServicesDir "frontend"
$frontendLog  = Join-Path $LogsDir "frontend.log"
$frontendErr  = Join-Path $LogsDir "frontend.err.log"

$existing = Get-Job -Name "frontend" -ErrorAction SilentlyContinue
if ($existing -and $existing.State -eq "Running") {
    Write-Warn "frontend is already running — skipping"
} else {
    if ($existing) { Remove-Job -Job $existing -Force }
    $frontendJob = Start-Job -Name "frontend" -ScriptBlock {
        param($Dir, $LogFile, $ErrFile)
        Set-Location $Dir
        & yarn dev 2>> $ErrFile >> $LogFile
    } -ArgumentList $frontendDir, $frontendLog, $frontendErr

    Write-OK "frontend started (job #$($frontendJob.Id)) — logs: .logs/frontend.log"
}

# ─────────────────────────────────────────────────────────────────────────────
# 8. Status summary
# ─────────────────────────────────────────────────────────────────────────────

Start-Sleep -Seconds 2   # give jobs a moment to fail fast if misconfigured

Write-Host "`n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" -ForegroundColor Cyan
Write-Host "  Risk Platform — Startup Summary" -ForegroundColor Cyan
Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" -ForegroundColor Cyan

$allJobs = Get-Job | Where-Object { $_.Name -in ($Jobs.Keys + @("frontend")) }
foreach ($j in ($allJobs | Sort-Object Name)) {
    $state  = $j.State
    $colour = if ($state -eq "Running") { "Green" } elseif ($state -eq "Failed") { "Red" } else { "Yellow" }
    $icon   = if ($state -eq "Running") { "  " } elseif ($state -eq "Failed") { "  " } else { "  " }
    Write-Host ("  {0,-22} {1}" -f $j.Name, $state) -ForegroundColor $colour
}

Write-Host "`n  Infrastructure:" -ForegroundColor Cyan
Write-Host "    Keycloak    http://localhost:8080           (admin/admin)" -ForegroundColor Gray
Write-Host "    Postgres    localhost:5432                  (postgres/postgres)" -ForegroundColor Gray
Write-Host "    Redis       localhost:6379" -ForegroundColor Gray
Write-Host "    Kafka       localhost:9092" -ForegroundColor Gray
Write-Host "    Jaeger      http://localhost:16686" -ForegroundColor Gray
Write-Host "    Prometheus  http://localhost:9090" -ForegroundColor Gray
Write-Host "    Grafana     http://localhost:3000           (admin/admin)" -ForegroundColor Gray

Write-Host "`n  Application:" -ForegroundColor Cyan
Write-Host "    Frontend    http://localhost:5173" -ForegroundColor Gray
Write-Host "    GraphQL     http://localhost:8090/graphql" -ForegroundColor Gray

Write-Host "`n  Useful commands:" -ForegroundColor Cyan
Write-Host "    Get-Job                                     # list all jobs" -ForegroundColor Gray
Write-Host "    Receive-Job -Name decision -Keep            # view service logs" -ForegroundColor Gray
Write-Host "    Get-Content .logs/decision.log -Tail 50 -Wait  # tail log file" -ForegroundColor Gray
Write-Host "    pwsh ./stop.ps1                             # stop everything" -ForegroundColor Gray
Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━`n" -ForegroundColor Cyan
