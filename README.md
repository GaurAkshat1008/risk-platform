# Payment Risk Platform

A multi-tenant, real-time payment risk decisioning platform built on Go, gRPC, GraphQL, and React.

Every inbound payment event is validated, deduplicated, evaluated against configurable tenant rule sets, assigned a risk decision (approve / flag / review / block), and surfaced to analysts, merchants, and downstream systems — all within a 150 ms p95 latency budget.

---

## Stack

| Layer        | Technologies                                                                            |
| ------------ | --------------------------------------------------------------------------------------- |
| **Backend**  | Go 1.25 · gRPC · PostgreSQL (pgx/v5) · Redis (go-redis/v9) · Kafka (segmentio/kafka-go) |
| **API**      | GraphQL (gqlgen) · gRPC                                                                 |
| **Frontend** | React 18 · Vite 6 · TypeScript 5.7 · Ant Design 5 · Apollo Client 3                     |
| **Auth**     | Keycloak 26 · OIDC / JWKS · JWT                                                         |
| **Infra**    | Docker · Kubernetes · APISIX                                                            |
| **Obs.**     | OpenTelemetry · Prometheus · Grafana · Jaeger                                           |

---

## Services

| #   | Service           | Port  | Description                                               |
| --- | ----------------- | ----- | --------------------------------------------------------- |
| 1   | Identity Access   | 50051 | JWT validation (Keycloak OIDC), RBAC, tenant isolation    |
| 2   | Tenant Config     | 50052 | Tenant onboarding, rule config, feature flags             |
| 3   | Ingestion         | 50053 | Payment event intake, dedup, idempotency, Kafka publish   |
| 4   | Risk Orchestrator | 50054 | Coordinates risk pipeline, enforces latency budget        |
| 5   | Rules Engine      | 50055 | Per-tenant composable JSON rule evaluation                |
| 6   | Decision          | 50056 | Final decision state machine, analyst overrides           |
| 7   | Case Management   | 50057 | Case lifecycle, assignments, SLA tracking                 |
| 8   | Workflow          | 50058 | Tenant workflow templates, state transition guards        |
| 9   | Audit Trail       | 50059 | Append-only audit log with cryptographic hash chain       |
| 10  | Explanation       | 50060 | Human-readable decision rationale                         |
| 11  | Notification      | 50061 | Async fanout via email, webhook, Slack                    |
| 12  | Log Ingestion     | 50062 | Structured log collection via OTel pipeline               |
| 13  | Ops Query         | 50063 | Faceted search, SLO tracking, alert feed                  |
| 14  | GraphQL BFF       | 8090  | Single API surface for the React UI                       |
| 15  | Frontend          | 5173  | Multi-portal React SPA (Merchant · Analyst · Ops · Admin) |

---

## Architecture

```
External Payment Source
        │ gRPC
        ▼
┌─────────────────┐
│   Ingestion     │──► payments.received (Kafka)
└─────────────────┘
                              │
                              ▼
                 ┌────────────────────────┐
                 │   Risk Orchestrator    │──► Rules Engine (gRPC)
                 └────────────────────────┘
                              │
                    risk.evaluated (Kafka)
                              │
                              ▼
                 ┌────────────────────────┐
                 │   Decision Service     │──► decision.made (Kafka)
                 └────────────────────────┘
                              │
               ┌──────────────┼──────────────┐
               ▼              ▼              ▼
        Case Management   Audit Trail   Notification

        ┌─────────────────────────────────┐
        │  Identity Access  :50051        │  ← every service validates here
        │  Tenant Config    :50052        │  ← every service reads config here
        └─────────────────────────────────┘

        ┌─────────────────────────────────┐
        │  GraphQL BFF  :8090             │  ← single API origin for React UI
        └─────────────────────────────────┘
                 │
                 ▼
        ┌─────────────────────────────────┐
        │  React Frontend  :5173          │
        └─────────────────────────────────┘
```

---

## Kafka Topics

| Topic               | Producer          | Consumers                                    |
| ------------------- | ----------------- | -------------------------------------------- |
| `payments.received` | Ingestion         | Risk Orchestrator                            |
| `risk.evaluated`    | Risk Orchestrator | Decision Service                             |
| `decision.made`     | Decision Service  | Case Management · Audit Trail · Notification |
| `rules.evaluated`   | Rules Engine      | Audit Trail                                  |
| `tenant.events`     | Tenant Config     | All services (config reload)                 |
| `auth.events`       | Identity Access   | Audit Trail                                  |
| `case.created`      | Case Management   | Notification · Audit Trail                   |
| `case.escalated`    | Case Management   | Notification · Audit Trail                   |
| `ops.logs`          | All services      | Log Ingestion                                |

---

## Infrastructure Services

| Service        | Port  | Purpose                        |
| -------------- | ----- | ------------------------------ |
| Keycloak       | 8080  | Identity provider, OIDC/JWT    |
| PostgreSQL     | 5432  | Per-service relational storage |
| Redis          | 6379  | Cache, rate limiting, locks    |
| Kafka          | 9092  | Async event bus (KRaft mode)   |
| OTel Collector | 4317  | Telemetry pipeline (OTLP gRPC) |
| Jaeger         | 16686 | Distributed tracing UI         |
| Prometheus     | 9090  | Metrics collection             |
| Grafana        | 3000  | Dashboards and alerting        |

---

## Prerequisites

- [Docker](https://docs.docker.com/get-docker/) + Docker Compose
- [Go 1.24+](https://go.dev/dl/)
- [Node 20+](https://nodejs.org/) + [Yarn](https://yarnpkg.com/)
- [PowerShell 7+](https://github.com/PowerShell/PowerShell) (`pwsh`)

---

## Quick Start

```powershell
# Full startup — infra + all services + frontend
pwsh ./start.ps1

# Infrastructure only (Docker Compose)
pwsh ./start.ps1 -InfraOnly

# Services only (infra already running)
pwsh ./start.ps1 -SkipInfra

# Use existing .env files as-is
pwsh ./start.ps1 -SkipEnvSetup
```

Once started:

| URL                              | Description            |
| -------------------------------- | ---------------------- |
| http://localhost:5173            | React frontend         |
| http://localhost:8090/playground | GraphQL playground     |
| http://localhost:8080            | Keycloak admin console |
| http://localhost:3000            | Grafana dashboards     |
| http://localhost:16686           | Jaeger trace explorer  |
| http://localhost:9090            | Prometheus             |

```powershell
# Stop all services (keep Docker running)
pwsh ./stop.ps1

# Stop all services + tear down Docker infra
pwsh ./stop.ps1 -Force
```

---

## Roles

| Role             | Permissions                                                 |
| ---------------- | ----------------------------------------------------------- |
| `platform_admin` | Unrestricted cross-tenant access (`*`)                      |
| `tenant_admin`   | `case:read/write` · `workflow:read/write` · `decision:read` |
| `analyst`        | `case:read/write` · `decision:read`                         |
| `merchant_user`  | `transaction:read` · `decision:read`                        |
| `ops_admin`      | `ops:read` · `audit:read`                                   |

---

## Key Design Principles

| Principle               | Implementation                                                                                                                     |
| ----------------------- | ---------------------------------------------------------------------------------------------------------------------------------- |
| **Multi-tenancy**       | Every record carries `tenant_id`; all DB queries, cache keys, and Kafka payloads are tenant-scoped                                 |
| **Fail-open safety**    | When upstream dependencies are unavailable, the pipeline defaults to `approve` with `fail_open=true` rather than blocking payments |
| **Idempotency**         | Every write path uses `ON CONFLICT DO NOTHING` + Redis dedup caches                                                                |
| **Observability first** | All services ship OTel traces, Prometheus metrics, and structured JSON logs                                                        |
| **Auto-migration**      | Services embed SQL via `//go:embed migrations/*.sql` and migrate on startup — no external tooling required                         |
| **Event-driven**        | Services communicate asynchronously via Kafka; gRPC is the secondary synchronous path                                              |

---

## Observability

Each Go service exposes:

- **Prometheus metrics** on a dedicated metrics port (`service_port + 39`, e.g. `:9090` for Identity Access)
- **OTel traces** exported via OTLP gRPC to the collector at `:4317`
- **Structured JSON logs** with `tenant_id`, `trace_id`, and operation-specific fields on every request

The GraphQL BFF tracks `QueryTotal`, `MutationTotal`, `CacheHits`, `CacheMisses`, and `RequestDuration`.

---

## Project Layout

```
.
├── start.ps1               # Full-stack startup script
├── stop.ps1                # Shutdown script
├── infra/
│   ├── docker/             # Docker Compose + config for all infra services
│   └── k8s/                # Kubernetes manifests (namespace, per-service)
└── services/
    ├── identity-access/    # :50051 — JWT/RBAC
    ├── tenant-config/      # :50052 — tenant management
    ├── ingestion/          # :50053 — payment intake
    ├── risk-orchestrator/  # :50054 — pipeline coordinator
    ├── rules-engine/       # :50055 — rule evaluation
    ├── decision/           # :50056 — final decision + overrides
    ├── case-management/    # :50057 — analyst case lifecycle
    ├── workflow/           # :50058 — workflow templates
    ├── audit-trail/        # :50059 — immutable audit log
    ├── explanation/        # :50060 — decision rationale
    ├── notification/       # :50061 — async alerts
    ├── log-ingestion/      # :50062 — log collection
    ├── ops-query/          # :50063 — ops search + SLO
    ├── graphql-bff/        # :8090  — GraphQL API gateway
    └── frontend/           # :5173  — React SPA
```

Each Go service follows the same layout:

```
service/
├── cmd/main.go             # Bootstrap: config → telemetry → DB → gRPC server
├── api/proto/              # Protobuf definitions
├── api/gen/                # Generated gRPC stubs
├── internal/
│   ├── config/             # Env-based config loader
│   ├── db/                 # pgx store layer
│   ├── cache/              # Redis cache wrappers
│   ├── kafka/              # Producer / consumer
│   ├── grpc/               # gRPC service handlers
│   └── telemetry/          # OTel tracer, Prometheus metrics
├── migrations/             # Embedded SQL migration files
└── Dockerfile              # Multi-stage scratch build, non-root user
```
