# Phase 3 — Workspaces + Infrastructure

Build Phase 3 from `docs/SPEC.md` — workspaces, infrastructure resources, and Terraform-based provisioning.

## Prerequisites

Phase 2 is complete. Connections, encryption, the provider interface, AWS validation, and the job system are working.

## Deliverables

This is the largest phase. Split into three sub-phases.

---

## Phase 3a — Workspaces + Resource Definitions (Week 1)

### Database Migrations

Create migrations for (see `docs/SPEC.md` Section 3.2):

1. `workspaces` — id, tenant_id, connection_id (FK to connections), name (unique per tenant), display_name, region, type (development/staging/production), label_color, cluster_config (JSONB), status, created_by, timestamps
2. `workspace_members` — id, workspace_id, user_id, role, timestamps. Unique on (workspace_id, user_id)
3. `resources` — id, tenant_id, workspace_id, name (unique per workspace), type, status, arguments (JSONB), attributes (JSONB), secret_attributes (BYTEA encrypted), provider_id, imported, created_by, timestamps
4. `resource_dependencies` — resource_id, depends_on_id, composite PK
5. `operations` — id, tenant_id, target_id, target_type, action, status, details (JSONB), error_message, triggered_by, started_at, finished_at, created_at

Also: add the FK from `api_keys.workspace_id` to `workspaces(id)` now that the workspaces table exists.

### Repository Layer

Write sqlx repository functions for all CRUD on workspaces, workspace_members, resources, resource_dependencies, and operations.

### Workspace Service Logic (`internal/services/workspace/`)

1. `Create` — validate name uniqueness per tenant, validate connection_id belongs to the same tenant, set status to "active"
2. `List` — all workspaces for the tenant, with member count
3. `Get` — workspace details
4. `Update` — update display_name, type, label_color, cluster_config
5. `Delete` — only if no resources or applications exist (fail with clear error if not empty)
6. `AddMember` / `RemoveMember` / `UpdateMemberRole` — workspace-level role overrides

### Resource Service Logic (`internal/services/infrastructure/`)

1. `Create` — validate name uniqueness per workspace, validate resource type is known, validate arguments against the type's argument definitions, store resource with status "pending"
2. `List` — all resources in a workspace, filterable by type and status
3. `Get` — resource with arguments, attributes, dependencies, latest operation
4. `Update` — update arguments (only when status is "active" or "failed")
5. `Delete` — enqueue destroy job, set status to "destroying"
6. `GetOperations` — operation history for a resource

### AWS Provider — Resource Types

Implement `GetResourceTypes()` and `GetDefaultArguments()` for these resource types:

1. **network** (VPC) — arguments: cidr_block, availability_zones, enable_nat_gateway, enable_dns
2. **kubernetes** (EKS) — arguments: cluster_version, endpoint_public_access, network_resource_id (dependency)
3. **node_group** — arguments: instance_types, min_size, max_size, desired_size, disk_size, kubernetes_resource_id (dependency)
4. **database** (RDS) — arguments: engine (postgres/mysql), engine_version, instance_class, allocated_storage, multi_az, network_resource_id
5. **cache** (ElastiCache) — arguments: engine (redis/memcached), node_type, num_cache_nodes, network_resource_id
6. **storage** (S3) — arguments: versioning_enabled, encryption

### Controllers (Echo route groups)

#### Workspace endpoints (`/api/v1/organizations/{org_id}/workspaces`)

- `POST /` — create workspace
- `GET /` — list workspaces
- `GET /{ws_id}` — get workspace
- `PUT /{ws_id}` — update workspace
- `DELETE /{ws_id}` — delete workspace (fails if not empty)

#### Resource endpoints (`/api/v1/organizations/{org_id}/workspaces/{ws_id}/resources`)

- `POST /` — create resource
- `GET /` — list resources (query params: `type`, `status`)
- `GET /{res_id}` — get resource with attributes and operations
- `PUT /{res_id}` — update resource arguments
- `DELETE /{res_id}` — trigger destroy
- `GET /{res_id}/operations` — list operations
- `GET /types` — list available resource types with argument definitions

### Workspace Middleware

Add workspace-scoped middleware that:
1. Extracts `ws_id` from URL
2. Validates the workspace belongs to the current tenant
3. Sets workspace context on the Echo context
4. Optionally checks workspace-level role (if workspace_members entry exists, use that role; otherwise fall back to org role)

### Frontend

1. **Workspace list** (`/orgs/{org_id}`) — show workspaces as cards with name, type, region, resource count, status color
2. **Create workspace modal** — name, display name, type selector, region, connection selector (from existing connections)
3. **Workspace detail page** (`/orgs/{org_id}/workspaces/{ws_id}`) — tabbed layout with Resources tab (active by default) and Settings tab
4. **Resource list** — table showing name, type, status, last operation time. Filterable by type.
5. **Create resource page** — select resource type, dynamic form generated from argument definitions, dependency selector (pick existing resources)
6. **Resource detail page** — shows arguments, attributes (with copy button), status, dependency graph, operation history timeline

### Tests

1. Workspace CRUD with tenant isolation
2. Resource creation with argument validation (missing required arg → error, invalid type → error)
3. Resource dependency validation (can't depend on resource in different workspace)
4. Workspace deletion blocked when resources exist
5. Resource type listing returns correct definitions for AWS

---

## Phase 3b — Provisioning Engine (Week 2)

### AWS Provider — Terraform Generation

Implement `GenerateTerraform(resource, dependencies)` for each resource type. This method generates HCL that:

1. Uses the AWS provider with credentials passed via environment variables
2. References dependency outputs (e.g., EKS module references VPC module outputs)
3. Includes a `terraform { backend "local" {} }` block (state stored locally per resource)
4. Outputs key attributes (VPC ID, cluster endpoint, database endpoint, etc.)
5. Tags resources with `kapstan:resource_id` for cost tracking

### Provisioning Job (`internal/jobs/`)

New job type: `provision_resource`

1. Load resource and its dependencies from DB
2. Decrypt the workspace connection's credentials
3. Call `provider.GenerateTerraform()` to get HCL
4. Write HCL to a persistent directory (`KAPSTAN_DATA_DIR/resources/{resource_id}/`)
5. Run `tofu init` (or `terraform init`)
6. Run `tofu apply -auto-approve` with AWS credentials as environment variables
7. Parse `tofu output -json` to extract attributes
8. Store attributes in the resource row (plain attributes in JSONB, secret attributes encrypted in BYTEA)
9. Update resource status to "active" on success, "failed" on error
10. Create an operation record with the outcome

`KAPSTAN_DATA_DIR` defaults to `/var/lib/kapstan/`. The Terraform state file is stored alongside the HCL — this state is needed for upgrades and destroys.

### Destroy Job

New job type: `destroy_resource`

1. Run `tofu destroy -auto-approve` against existing state
2. Clean up state directory
3. Update resource status to "destroyed"
4. Create operation record

### Resource Status State Machine

```
pending → provisioning → active
                       → failed → provisioning (retry)
active → provisioning (upgrade)
active → destroying → destroyed
                    → failed
```

### Controller Additions

- `POST .../resources/{res_id}/provision` — trigger provisioning job
- `POST .../resources/{res_id}/retry` — retry failed provisioning

### Tests

1. **Terraform generation** — for each resource type, verify generated HCL is valid
2. **Provisioning job** — mock `tofu` execution, verify state transitions and attribute storage
3. **Dependency ordering** — resource with unprovisioned dependency → error
4. **Destroy job** — verify cleanup and status transition

---

## Phase 3c — Resource Management (Week 3)

### AWS Provider — Metrics

Implement `GetResourceMetrics()` for key resource types via CloudWatch:
- database: CPUUtilization, DatabaseConnections, FreeStorageSpace, ReadIOPS, WriteIOPS
- cache: CPUUtilization, CacheHits, CacheMisses, CurrConnections
- kubernetes: cluster status via EKS DescribeCluster API

### AWS Provider — Temporary Credentials

Implement `GenerateTemporaryCredentials()`:
- For database resources — generate a short-lived IAM auth token or return stored credentials
- For S3 — generate pre-signed URLs or temporary STS credentials

### Controller Additions

- `GET .../resources/{res_id}/metrics?metric={name}&start={iso}&end={iso}&period={seconds}`
- `GET .../resources/{res_id}/metrics/options` — available metric names for this resource type
- `GET .../resources/{res_id}/credentials` — generate temporary access credentials
- `POST .../resources/import` — create resource record for existing cloud resource (status = "active", no Terraform state)

### Frontend Additions

1. Resource detail — metrics tab with line charts and time range selector
2. Resource detail — credentials button with auto-expire warning
3. Provision button on pending resources
4. Operation history timeline with status and duration
5. Resource status badges with auto-refresh

### Tests

1. Metrics endpoint returns data points in expected format (mock CloudWatch)
2. Temporary credentials are generated correctly (mock STS/RDS)
3. Import creates resource without triggering provisioning
4. Full lifecycle: create → provision → upgrade (change args) → destroy

## Constraints

- Terraform state files must be stored persistently — losing state means losing the ability to manage the resource
- Never log credentials, even at debug level
- Resource operations must be idempotent
- The `tofu`/`terraform` binary must be on PATH; Kapstan should check for its presence at startup and log a warning if missing (not fatal — it's only needed for infrastructure features)
