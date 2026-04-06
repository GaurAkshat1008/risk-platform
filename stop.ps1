#!/usr/bin/env pwsh
<#
.SYNOPSIS
    Stops all Risk Platform services and optionally tears down Docker infrastructure.

.PARAMETER KeepInfra
    Stop application services only; leave Docker Compose running.

.PARAMETER Force
    Kill Go and Node processes by port scan in addition to stopping jobs.

.EXAMPLE
    pwsh ./stop.ps1               # Stop services + keep Docker running
    pwsh ./stop.ps1 -KeepInfra   # Same — explicit flag
    pwsh ./stop.ps1 -Force        # Also kill stray processes by port
#>
param(
    [switch]$KeepInfra,
    [switch]$Force
)

$Root     = $PSScriptRoot
$InfraDir = Join-Path $Root "infra" "docker"

function Write-Step { param($Msg) Write-Host "`n==> $Msg" -ForegroundColor Cyan }
function Write-OK   { param($Msg) Write-Host "    [ OK ] $Msg" -ForegroundColor Green }
function Write-Warn { param($Msg) Write-Host "    [WARN] $Msg" -ForegroundColor Yellow }
function Write-Info { param($Msg) Write-Host "           $Msg" -ForegroundColor Gray }

# ─────────────────────────────────────────────────────────────────────────────
# 1. Stop PowerShell background jobs
# ─────────────────────────────────────────────────────────────────────────────

Write-Step "Stopping background jobs"

$serviceNames = @(
    "identity-access", "tenant-config", "ingestion", "risk-orchestrator",
    "rules-engine", "decision", "case-management", "workflow", "audit-trail",
    "explanation", "notification", "log-ingestion", "ops-query", "graphql-bff",
    "frontend"
)

$found = 0
foreach ($name in $serviceNames) {
    $job = Get-Job -Name $name -ErrorAction SilentlyContinue
    if ($job) {
        Stop-Job  -Job $job
        Remove-Job -Job $job -Force
        Write-OK "Stopped job: $name"
        $found++
    }
}

if ($found -eq 0) {
    Write-Warn "No matching jobs found. Services may have been started outside this script."
}

# ─────────────────────────────────────────────────────────────────────────────
# 2. (Optional) Kill stray processes by port
# ─────────────────────────────────────────────────────────────────────────────

if ($Force) {
    Write-Step "Killing stray processes on service ports (50051-50063, 8090, 5173)"

    $ports = @(50051, 50052, 50053, 50054, 50055, 50056, 50057, 50058, 50059, 50060, 50061, 50062, 50063, 8090, 5173)

    foreach ($port in $ports) {
        if ($IsWindows) {
            $lines = netstat -ano 2>$null | Select-String ":$port\s"
            foreach ($line in $lines) {
                if ($line -match '\s+(\d+)$') {
                    $pid = [int]$Matches[1]
                    try {
                        Stop-Process -Id $pid -Force -ErrorAction SilentlyContinue
                        Write-OK "Killed PID $pid (port $port)"
                    } catch {}
                }
            }
        } else {
            # Linux / macOS
            $pid = (lsof -ti ":$port" 2>/dev/null)
            if ($pid) {
                try {
                    Stop-Process -Id ([int]$pid) -Force -ErrorAction SilentlyContinue
                    Write-OK "Killed PID $pid (port $port)"
                } catch {}
            }
        }
    }
}

# ─────────────────────────────────────────────────────────────────────────────
# 3. Docker Compose (unless -KeepInfra)
# ─────────────────────────────────────────────────────────────────────────────

if (-not $KeepInfra) {
    Write-Step "Stopping Docker Compose infrastructure"
    Push-Location $InfraDir
    try {
        docker compose down 2>&1 | Out-Null
        Write-OK "docker compose down complete"
    } catch {
        Write-Warn "docker compose down failed: $_"
    } finally {
        Pop-Location
    }
} else {
    Write-Warn "-KeepInfra set — Docker Compose left running"
}

# ─────────────────────────────────────────────────────────────────────────────
# 4. Summary
# ─────────────────────────────────────────────────────────────────────────────

Write-Host "`n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" -ForegroundColor Cyan
Write-Host "  Platform stopped." -ForegroundColor Cyan
if ($KeepInfra) {
    Write-Host "  Docker infra is still running — use 'docker compose down' in infra/docker/ to stop it." -ForegroundColor Gray
}
Write-Host "  Logs are preserved in .logs/" -ForegroundColor Gray
Write-Host "  Run 'pwsh ./start.ps1 -SkipEnvSetup' to restart." -ForegroundColor Gray
Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━`n" -ForegroundColor Cyan
