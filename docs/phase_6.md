# Phase 6 — Cost + Monitoring

Build Phase 6 from `docs/SPEC.md` — cost tracking, resource metrics, application metrics.

## Prerequisites

Phase 5 is complete. Applications deploy to Kubernetes, the external deploy API works, notifications are delivered.

## Deliverables

### Database Migrations

```sql
CREATE TABLE cost_records (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES organizations(id),
    workspace_id    UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    target_id       UUID NOT NULL,
    target_type     TEXT NOT NULL,           -- 'resource' or 'application'
    date            DATE NOT NULL,
    amount_cents    BIGINT NOT NULL,         -- cost in cents (avoids float issues)
    currency        TEXT NOT NULL DEFAULT 'USD',
    breakdown       JSONB,                  -- {"cpu": 1200, "memory": 800, "storage": 500}
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(target_id, target_type, date)
);

CREATE TABLE cost_settings (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES organizations(id),
    connection_id   UUID REFERENCES connections(id),
    enabled         BOOLEAN NOT NULL DEFAULT false,
    last_synced_at  TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(tenant_id)
);

CREATE TABLE application_events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    application_id  UUID NOT NULL REFERENCES applications(id) ON DELETE CASCADE,
    event_type      TEXT NOT NULL,       -- 'normal', 'warning'
    reason          TEXT NOT NULL,       -- 'Pulling', 'Started', 'BackOff', 'OOMKilled'
    message         TEXT,
    source          TEXT,
    first_seen      TIMESTAMPTZ,
    last_seen       TIMESTAMPTZ,
    count           INT DEFAULT 1,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_cost_records_workspace ON cost_records(workspace_id, date);
CREATE INDEX idx_cost_records_tenant ON cost_records(tenant_id, date);
CREATE INDEX idx_app_events ON application_events(application_id, created_at DESC);
```

### AWS Provider — Cost

Implement `GetCostBreakdown(ctx, filters)`:
1. Use AWS Cost Explorer API (`GetCostAndUsage`)
2. Filter by date range, group by resource tags
3. Map AWS resource tags back to Kapstan resource/application IDs (resources are tagged with `kapstan:resource_id` from Phase 3b)

### Cost Service Logic (`internal/services/cost/`)

1. `SyncCosts(tenantID)` — pull cost data from cloud provider, upsert into `cost_records`
2. `GetWorkspaceCosts(workspaceID, start, end)` — aggregate costs for all resources and apps
3. `GetResourceCosts(resourceID, start, end)` — cost history for a single resource
4. `GetApplicationCosts(appID, start, end)` — cost history for a single application
5. `GetTenantCostSummary(tenantID, start, end)` — total across all workspaces, grouped by workspace
6. `EnableCostTracking(tenantID, connectionID)` / `DisableCostTracking(tenantID)`

### Cost Sync Job

Periodic job: `cost_sync` — runs daily at 2am UTC. For each tenant with cost tracking enabled, pull last 3 days of cost data (to catch delayed billing).

### Application Metrics

Query K8s Metrics API (`metrics.k8s.io/v1beta1`) via client-go:
- `cpu_usage` — CPU across all pods
- `memory_usage` — memory across all pods
- `pod_count` — running pods over time
- `restart_count` — container restarts over time

If Prometheus is available in the cluster (detected by checking for the service), query it for historical data. Otherwise, only current metrics.

### Application Events

Periodic job: `sync_app_events` — every 60 seconds for active applications, poll K8s events, upsert into `application_events` table.

Trigger notifications for:
- `pod_crash_loop` — when BackOff event detected
- `pod_oom_killed` — when OOMKilled event detected

### Prometheus `/metrics` Endpoint

Expose `GET /metrics` with Prometheus-format metrics about Kapstan itself:
- `kapstan_http_requests_total` (counter by method, path, status)
- `kapstan_http_request_duration_seconds` (histogram)
- `kapstan_jobs_total` (counter by type, status)
- `kapstan_jobs_duration_seconds` (histogram)
- `kapstan_active_deployments` (gauge)
- `kapstan_resources_total` (gauge by type, status)

Use `prometheus/client_golang`. Add Echo middleware for HTTP instrumentation.

### Controllers

#### Cost
- `GET /api/v1/organizations/{org_id}/costs` — tenant cost summary (query: start, end, group_by=workspace)
- `GET .../workspaces/{ws_id}/costs` — workspace costs (query: start, end, group_by=resource|application)
- `GET .../resources/{res_id}/costs` — resource cost history
- `GET .../applications/{app_id}/costs` — application cost history
- `GET /api/v1/organizations/{org_id}/cost-settings` — get settings
- `PUT /api/v1/organizations/{org_id}/cost-settings` — enable/disable

#### Metrics
- `GET .../resources/{res_id}/metrics` — resource metrics (query: metric, start, end, period)
- `GET .../resources/{res_id}/metrics/options` — available metrics
- `GET .../applications/{app_id}/metrics` — application metrics
- `GET .../applications/{app_id}/metrics/options` — available metrics
- `GET .../applications/{app_id}/events` — application events (query: start, end, limit)

### Frontend

1. **Cost dashboard** (`/orgs/{org_id}/costs`) — summary cards, daily cost bar chart, top resources/apps by cost
2. **Workspace cost tab** — cost breakdown by resource and application
3. **Resource cost section** on resource detail — cost trend line chart
4. **Application cost section** on app detail — cost trend line chart
5. **Resource metrics tab** — metric selector + time range picker + line chart
6. **Application metrics tab** — CPU/memory usage charts, pod count, restart count
7. **Application events tab** — event table with type badges, auto-refresh every 30s
8. **Cost settings page** (`/orgs/{org_id}/settings/cost`) — enable/disable, select AWS connection

### Tests

1. Cost sync — mock AWS Cost Explorer, verify records created with correct amounts
2. Cost aggregation — create records, verify summaries aggregate correctly
3. Resource metrics — mock CloudWatch, verify data format
4. Application metrics — mock K8s metrics API, verify pod aggregation
5. Event tracking — mock K8s events, verify stored and notifications triggered
6. Prometheus endpoint — verify `/metrics` returns valid format

## Constraints

- Cost amounts stored as integer cents, not floating point
- Cost sync must be idempotent (upsert, no duplicates)
- Metrics endpoints max 30 day time range
- `/metrics` endpoint must not require authentication
- Don't store metrics in Kapstan's DB — query from source on demand. Only cost data is cached.
