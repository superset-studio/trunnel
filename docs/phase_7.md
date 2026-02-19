# Phase 7 — Polish + GCP Provider

Build Phase 7 from `docs/SPEC.md` — second cloud provider, workspace cloning, audit logging, production readiness.

## Prerequisites

Phase 6 is complete. The platform is functionally complete for AWS.

## Deliverables

---

## Phase 7a — GCP Provider (Week 1)

### GCP Provider (`internal/provider/gcp/`)

Implement the full `Provider` interface for Google Cloud Platform.

#### Credentials
GCP connection stores a service account JSON key (encrypted). Authenticate using this key.

#### ValidateCredentials + GetAccountInfo
Call Resource Manager API to verify access. Return project ID and service account email.

#### GetResourceTypes + GetDefaultArguments

1. **network** (VPC) — auto_create_subnetworks, routing_mode, region
2. **kubernetes** (GKE) — cluster_version, machine_type, initial_node_count, network_resource_id, region/zone
3. **node_group** (node pool) — machine_type, min_count, max_count, disk_size_gb, kubernetes_resource_id
4. **database** (Cloud SQL) — database_version, tier, disk_size, availability_type, network_resource_id
5. **cache** (Memorystore) — tier, memory_size_gb, redis_version, network_resource_id
6. **storage** (GCS) — location, storage_class, versioning

#### GenerateTerraform
Generate HCL for each GCP resource type using the Google Terraform provider.

#### GetKubeConfig
Call GKE API, generate kubeconfig with OAuth2 token auth.

#### ListRepositories + ListImageTags
Support GCP Artifact Registry.

#### GetResourceMetrics
Use GCP Cloud Monitoring API (Stackdriver).

#### GetCostBreakdown
Use GCP Billing Export (BigQuery) or Cloud Billing API.

### Frontend Updates
1. GCP option in connection creation (service account JSON file upload)
2. GCP resource types shown when workspace uses a GCP connection
3. All features work seamlessly via provider interface

### Tests
1. GCP credential validation (mock APIs)
2. Terraform generation for each GCP resource type
3. GKE kubeconfig generation
4. Full lifecycle with GCP mocks

---

## Phase 7b — Workspace Cloning + Managed Services (Week 1, parallel)

### Workspace Cloning

`POST /api/v1/organizations/{org_id}/workspaces/{ws_id}/clone`:
```json
{
  "name": "staging",
  "display_name": "Staging Environment",
  "type": "staging",
  "connection_id": "...",
  "region": "us-west-2"
}
```

Copies: resource definitions, applications + full config, workspace env vars, notification rules.
Does NOT copy: deployment history, operations, cost records, resource attributes/state (resources start as "pending").

Must be atomic — if any part fails, nothing is created (use a DB transaction).

### Managed Services

Cluster add-ons that Kapstan installs into K8s clusters.

#### Database Migration
```sql
CREATE TABLE managed_services (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES organizations(id),
    workspace_id    UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    cluster_id      UUID NOT NULL REFERENCES resources(id),
    name            TEXT NOT NULL,
    type            TEXT NOT NULL,     -- 'cert_manager', 'external_dns', 'metrics_server', 'ingress_nginx'
    status          TEXT NOT NULL DEFAULT 'pending',
    config          JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(workspace_id, type)
);
```

#### Supported Services (Phase 7)
1. **cert-manager** — TLS certificate management
2. **ingress-nginx** — NGINX ingress controller
3. **metrics-server** — K8s metrics API (required for HPA)
4. **external-dns** — DNS record management

Each installed via its official Helm chart.

#### Controllers (`.../workspaces/{ws_id}/managed-services`)
- `POST /`, `GET /`, `GET /{svc_id}`, `DELETE /{svc_id}`

#### Frontend
Managed services tab on workspace page — grid of available services with install/uninstall, status badges.

---

## Phase 7c — Audit Log + Production Polish (Week 2)

### Audit Log

```sql
CREATE TABLE audit_log (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES organizations(id),
    actor_id        UUID REFERENCES users(id),
    actor_email     TEXT NOT NULL,
    action          TEXT NOT NULL,
    resource_type   TEXT NOT NULL,
    resource_id     UUID,
    resource_name   TEXT,
    details         JSONB,
    ip_address      TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_audit_tenant ON audit_log(tenant_id, created_at DESC);
```

Add audit logging to every state-changing API call. Never log secrets in audit details.

#### Controller
- `GET /api/v1/organizations/{org_id}/audit-log` — paginated, filterable by action, resource_type, actor, date range

#### Frontend
Audit log page (`/orgs/{org_id}/settings/audit-log`) — table with filters and pagination.

### Graceful Shutdown

On SIGTERM: stop accepting requests, wait for in-flight (30s), stop job runner, close DB, exit.

### Health Check Improvements

- `/healthz` — always 200 (liveness)
- `/readyz` — 200 if DB reachable and job runner healthy; 503 during shutdown

### Configuration Validation

On startup, validate all config:
- `DATABASE_URL` is a valid PostgreSQL connection string
- `KAPSTAN_ENCRYPTION_KEY` is exactly 32 bytes (64 hex chars)
- Warn if `tofu`/`terraform` not on PATH
- Warn if `REDIS_URL` set but unreachable

### Helm Chart (`deploy/helm/kapstan/`)

Production-ready Helm chart for self-hosted installation with configurable values: image, replicas, resources, PostgreSQL connection, Redis, encryption key, ingress, service account.

### Documentation

Update `README.md`: what Kapstan is, quick start (Docker), full install (Helm), config reference, architecture overview, API overview, contributing guide.

### Frontend Polish

1. Global error boundary + toast notifications
2. Skeleton loaders for data-fetching pages
3. Empty states with CTAs when no data exists
4. Responsive sidebar navigation + breadcrumbs
5. Settings consolidation (members, API keys, connections, cost, notifications, audit log)

### Tests

1. Workspace cloning — verify deep copy, new IDs, no history copied, atomic (partial failure → nothing created)
2. Managed services — mock Helm, verify install/uninstall/status
3. Audit log — perform actions, verify records with correct details
4. GCP full lifecycle — end-to-end with mocked APIs
5. Graceful shutdown — in-flight requests complete after SIGTERM
6. Helm chart — `helm template` produces valid K8s manifests
7. Config validation — invalid encryption key → clear startup error

## Constraints

- Audit log must never contain secrets, passwords, or credentials
- Workspace cloning must be atomic
- GCP provider must not break AWS functionality
- Helm chart must support both bundled and external PostgreSQL
- All new features must have tenant isolation tests
