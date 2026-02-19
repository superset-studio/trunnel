# Kapstan Open Source — Technical Specification

## 1. Vision

A single Go binary + PostgreSQL that lets teams deploy applications to Kubernetes and provision cloud infrastructure through a web UI and REST API. Multi-tenant, multi-cloud (AWS first, then GCP/Azure), open source.

**Install experience:**

```bash
# Option A: Docker Compose
docker compose up  # starts backend, frontend, and PostgreSQL

# Option B: Helm
helm install kapstan kapstan/kapstan --set postgresql.enabled=true

# Option C: Binary + separate frontend
kapstan server --database-url postgres://...
```

Two services (backend API + frontend SPA). One database. Working platform.

---

## 2. Architecture

### 2.1 Modular Monolith

Single Go binary with service packages. No microservices, no gRPC, no protobuf codegen.

```
kapstan/
├── api/                             # Go backend (module: github.com/superset-studio/kapstan/api)
│   ├── cmd/kapstan/
│   │   └── main.go                  # single entry point: `kapstan server`
│   ├── internal/
│   │   ├── controllers/             # HTTP handlers, middleware, routing
│   │   │   ├── router.go
│   │   │   ├── middleware/          # auth, tenant scoping, RBAC, request logging
│   │   │   ├── organization/       # tenant/user/role controllers
│   │   │   ├── workspace/          # workspace controllers
│   │   │   ├── connection/         # cloud connection controllers
│   │   │   ├── infrastructure/     # resource controllers
│   │   │   ├── application/        # app/container/deployment controllers
│   │   │   ├── notification/       # notification rule controllers
│   │   │   └── external/           # external deploy API (API key auth)
│   │   ├── models/                  # shared domain model structs (imported by all layers)
│   │   ├── services/                # business logic — no HTTP, no SQL
│   │   │   ├── organization/       # tenant, user, role, API key logic
│   │   │   ├── workspace/          # workspace lifecycle
│   │   │   ├── connection/         # credential management, validation
│   │   │   ├── infrastructure/     # resource definitions, provisioning orchestration
│   │   │   ├── application/        # app config, container specs, deployment orchestration
│   │   │   ├── deployment/         # Helm/K8s deployment execution
│   │   │   ├── provisioning/       # Terraform/OpenTofu execution
│   │   │   ├── cost/               # cost aggregation
│   │   │   └── notification/       # notification dispatch
│   │   ├── provider/                # cloud provider abstraction
│   │   │   ├── provider.go         # interface definitions
│   │   │   ├── aws/
│   │   │   ├── gcp/                # phase 2
│   │   │   └── azure/              # phase 3
│   │   ├── jobs/                    # background job runner
│   │   │   ├── runner.go           # PostgreSQL-backed job queue
│   │   │   ├── scheduler.go        # periodic job scheduling
│   │   │   └── handlers.go         # job type handlers
│   │   ├── platform/                # cross-cutting infrastructure
│   │   │   ├── config/             # configuration loading (env vars, flags)
│   │   │   ├── database/           # PostgreSQL connection, migrations
│   │   │   ├── crypto/             # encryption for secrets at rest
│   │   │   ├── auth/               # JWT generation/validation, password hashing
│   │   │   └── logging/            # structured logging
│   │   └── repositories/            # data access layer (sqlx queries)
│   │       ├── tenant.go
│   │       ├── user.go
│   │       ├── workspace.go
│   │       ├── connection.go
│   │       ├── resource.go
│   │       ├── application.go
│   │       ├── deployment.go
│   │       ├── operation.go
│   │       ├── notification.go
│   │       └── job.go
│   ├── migrations/                  # SQL migration files (golang-migrate)
│   ├── go.mod
│   └── go.sum
├── web/                             # React frontend (standalone service)
│   ├── src/
│   ├── package.json
│   └── vite.config.ts
├── deploy/
│   ├── Dockerfile                   # Go backend Dockerfile
│   ├── Dockerfile.web               # Frontend Dockerfile
│   └── helm/
│       └── kapstan/                 # Helm chart for self-hosted install
├── docs/
│   └── SPEC.md                      # this file
├── Makefile                         # top-level orchestration
└── README.md
```

### 2.2 Key Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Architecture | Modular monolith | One container to deploy; extract services later if needed |
| API | REST + OpenAPI | curl-friendly, broad ecosystem, auto-generated client types |
| Database | PostgreSQL only | Proven, handles jobs/queues/JSON/encryption natively |
| Database access | sqlx | Thin wrapper over database/sql. Write SQL directly, scan into structs. No code generation step. |
| Job queue | PostgreSQL-backed (River) | No Temporal/Redis dependency. Durable, transactional with application data. |
| Auth | Built-in (bcrypt + JWT) | Zero external dependencies to get started |
| Secrets at rest | AES-256-GCM, key from env var | Works anywhere. Optional KMS backends later. |
| Frontend | Standalone React app | Separate service. Backend is a pure API server. Deployed independently. |
| Config | Environment variables | 12-factor. No config files required. |
| Migrations | golang-migrate | SQL files, version controlled, no ORM magic. |
| Cloud abstraction | Provider interface | One interface, implement per cloud. Services never import cloud SDKs. |
| IaC execution | Shelling out to `tofu` / `terraform` | Battle-tested. Don't reimplement Terraform in Go. |
| K8s deployment | Helm SDK (Go library) | Programmatic Helm installs, no shelling out |
| HTTP framework | Echo | High-performance, expressive API, built-in middleware, request binding/validation |
| Logging | slog (stdlib) | Standard library, structured, zero dependencies |

### 2.3 Dependencies (Runtime)

**Required:**
- PostgreSQL 15+

**Optional:**
- Redis (caching — falls back to in-memory LRU)
- `tofu` or `terraform` binary on PATH (for infrastructure provisioning)
- `helm` awareness via Go SDK (bundled)
- Kubeconfig or in-cluster config (for K8s deployments)

That's it. No Temporal, NATS, Elasticsearch, Vault, OPA, Jaeger.

---

## 3. Data Model

### 3.1 Multi-Tenancy

Row-level tenancy. Every table that holds tenant-scoped data has a `tenant_id` column. Middleware extracts tenant from JWT, sets it on the request context, and the repository layer scopes all queries.

No separate databases. No schema-per-tenant. A column and a middleware.

### 3.2 Core Entities

#### organizations

The top-level tenant. Maps to a company or team.

```sql
CREATE TABLE organizations (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL UNIQUE,          -- url-safe slug
    display_name TEXT NOT NULL,
    logo_url    TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

#### users

```sql
CREATE TABLE users (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email           TEXT NOT NULL UNIQUE,
    password_hash   TEXT,                       -- null for invited users who haven't set password
    name            TEXT NOT NULL,
    avatar_url      TEXT,
    email_verified  BOOLEAN NOT NULL DEFAULT false,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

#### organization_members

Links users to organizations with a role.

```sql
CREATE TABLE organization_members (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id),
    user_id         UUID NOT NULL REFERENCES users(id),
    role            TEXT NOT NULL DEFAULT 'member',   -- 'owner', 'admin', 'member', 'viewer'
    invited_by      UUID REFERENCES users(id),
    invited_at      TIMESTAMPTZ,
    accepted_at     TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(organization_id, user_id)
);
```

#### workspaces

An environment (dev, staging, production) within an organization.

```sql
CREATE TABLE workspaces (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES organizations(id),
    connection_id   UUID REFERENCES connections(id),   -- cloud account for this workspace
    name            TEXT NOT NULL,                      -- url-safe slug
    display_name    TEXT NOT NULL,
    region          TEXT,                               -- e.g., 'us-east-1'
    type            TEXT NOT NULL DEFAULT 'development', -- 'development', 'staging', 'production'
    label_color     TEXT,
    cluster_config  JSONB,                              -- kubeconfig reference or in-cluster details
    status          TEXT NOT NULL DEFAULT 'pending',     -- 'pending', 'active', 'deleting', 'failed'
    created_by      UUID REFERENCES users(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(tenant_id, name)
);
```

#### workspace_members

Optional workspace-level role overrides.

```sql
CREATE TABLE workspace_members (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id    UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    user_id         UUID NOT NULL REFERENCES users(id),
    role            TEXT NOT NULL DEFAULT 'member',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(workspace_id, user_id)
);
```

#### connections

Cloud provider credentials and third-party integrations.

```sql
CREATE TABLE connections (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES organizations(id),
    name            TEXT NOT NULL,
    category        TEXT NOT NULL,   -- 'aws', 'gcp', 'azure', 'github', 'slack', 'docker_hub', 'ecr', etc.
    status          TEXT NOT NULL DEFAULT 'pending', -- 'pending', 'valid', 'invalid', 'expired'
    last_validated  TIMESTAMPTZ,
    credentials     BYTEA NOT NULL,  -- AES-256-GCM encrypted JSON blob
    config          JSONB,           -- non-secret config (region, account ID, etc.)
    created_by      UUID REFERENCES users(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(tenant_id, name)
);
```

Credentials are encrypted at rest using AES-256-GCM. The encryption key comes from `KAPSTAN_ENCRYPTION_KEY` env var. The `credentials` column stores:

```json
{
    "access_key_id": "AKIA...",
    "secret_access_key": "...",
    "role_arn": "arn:aws:iam::..."
}
```

The shape varies by `category`. Decrypted only in the provider layer, never exposed via API.

#### api_keys

Programmatic access scoped to a workspace.

```sql
CREATE TABLE api_keys (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES organizations(id),
    workspace_id    UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    key_prefix      TEXT NOT NULL,          -- first 8 chars, for display ("kap_a1b2...")
    key_hash        TEXT NOT NULL UNIQUE,   -- bcrypt hash for lookup
    access_level    TEXT NOT NULL DEFAULT 'deploy', -- 'deploy', 'read', 'admin'
    created_by      UUID REFERENCES users(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_used_at    TIMESTAMPTZ
);
```

#### resources

Cloud infrastructure resources (VPCs, databases, clusters, etc.).

```sql
CREATE TABLE resources (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES organizations(id),
    workspace_id    UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    type            TEXT NOT NULL,   -- 'network', 'kubernetes', 'database', 'storage', 'cache', 'queue', 'certificate', 'node_group', 'custom'
    status          TEXT NOT NULL DEFAULT 'pending', -- 'pending', 'provisioning', 'active', 'failed', 'destroying', 'destroyed'
    arguments       JSONB NOT NULL DEFAULT '{}',     -- input params for provisioning (Terraform vars)
    attributes      JSONB NOT NULL DEFAULT '{}',     -- output values after provisioning (Terraform outputs)
    secret_attributes BYTEA,                          -- encrypted sensitive outputs
    provider_id     TEXT,                              -- cloud resource ID (ARN, etc.)
    imported        BOOLEAN NOT NULL DEFAULT false,
    created_by      UUID REFERENCES users(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(workspace_id, name)
);
```

#### resource_dependencies

DAG of infrastructure dependencies.

```sql
CREATE TABLE resource_dependencies (
    resource_id     UUID NOT NULL REFERENCES resources(id) ON DELETE CASCADE,
    depends_on_id   UUID NOT NULL REFERENCES resources(id) ON DELETE CASCADE,
    PRIMARY KEY (resource_id, depends_on_id)
);
```

#### applications

A deployable workload (set of containers on a K8s cluster).

```sql
CREATE TABLE applications (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES organizations(id),
    workspace_id    UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    cluster_id      UUID REFERENCES resources(id),     -- which K8s cluster to deploy to
    name            TEXT NOT NULL,
    display_name    TEXT NOT NULL,
    namespace       TEXT NOT NULL DEFAULT 'default',
    type            TEXT NOT NULL DEFAULT 'deployment', -- 'deployment', 'cronjob', 'chart', 'custom_chart'
    status          TEXT NOT NULL DEFAULT 'pending',    -- 'pending', 'deploying', 'running', 'failed', 'stopped'
    replicas        INT NOT NULL DEFAULT 1,
    cd_enabled      BOOLEAN NOT NULL DEFAULT false,
    imported        BOOLEAN NOT NULL DEFAULT false,
    created_by      UUID REFERENCES users(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(workspace_id, name)
);
```

#### containers

Container specs within an application.

```sql
CREATE TABLE containers (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    application_id  UUID NOT NULL REFERENCES applications(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    type            TEXT NOT NULL DEFAULT 'main',  -- 'main', 'sidecar', 'init'
    image_uri       TEXT NOT NULL,                 -- full image reference, e.g. '123456.dkr.ecr.us-east-1.amazonaws.com/app'
    image_tag       TEXT NOT NULL DEFAULT 'latest',
    connection_id   UUID REFERENCES connections(id), -- registry credentials
    pull_policy     TEXT NOT NULL DEFAULT 'IfNotPresent',
    command         TEXT[],
    args            TEXT[],
    cpu_request     TEXT,    -- '100m'
    cpu_limit       TEXT,    -- '500m'
    memory_request  TEXT,    -- '128Mi'
    memory_limit    TEXT,    -- '512Mi'
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(application_id, name)
);
```

#### container_ports

```sql
CREATE TABLE container_ports (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    container_id    UUID NOT NULL REFERENCES containers(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    container_port  INT NOT NULL,
    protocol        TEXT NOT NULL DEFAULT 'TCP',
    is_public       BOOLEAN NOT NULL DEFAULT false,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(container_id, name)
);
```

#### container_env_vars

```sql
CREATE TABLE container_env_vars (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    container_id    UUID NOT NULL REFERENCES containers(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    value           TEXT,            -- plaintext value (for non-secret)
    secret_value    BYTEA,           -- encrypted (for secret type)
    type            TEXT NOT NULL DEFAULT 'plain',  -- 'plain', 'secret'
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(container_id, name)
);
```

#### container_probes

```sql
CREATE TABLE container_probes (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    container_id            UUID NOT NULL REFERENCES containers(id) ON DELETE CASCADE,
    probe_type              TEXT NOT NULL,  -- 'liveness', 'readiness', 'startup'
    mechanism               TEXT NOT NULL,  -- 'http', 'tcp', 'exec'
    path                    TEXT,           -- for HTTP probes
    port                    INT,
    command                 TEXT[],         -- for exec probes
    initial_delay_seconds   INT NOT NULL DEFAULT 0,
    period_seconds          INT NOT NULL DEFAULT 10,
    timeout_seconds         INT NOT NULL DEFAULT 1,
    success_threshold       INT NOT NULL DEFAULT 1,
    failure_threshold       INT NOT NULL DEFAULT 3,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(container_id, probe_type)
);
```

#### container_volumes

```sql
CREATE TABLE container_volumes (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    container_id    UUID NOT NULL REFERENCES containers(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    mount_path      TEXT NOT NULL,
    size            TEXT,                               -- '10Gi'
    storage_class   TEXT,
    read_only       BOOLEAN NOT NULL DEFAULT false,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(container_id, name)
);
```

#### workspace_env_vars

Global environment variables shared across all containers in a workspace.

```sql
CREATE TABLE workspace_env_vars (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id    UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    value           TEXT,
    secret_value    BYTEA,
    type            TEXT NOT NULL DEFAULT 'plain',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(workspace_id, name)
);
```

#### ingresses

Load balancer / ingress configuration for applications.

```sql
CREATE TABLE ingresses (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    application_id      UUID NOT NULL REFERENCES applications(id) ON DELETE CASCADE,
    hostname            TEXT,
    path                TEXT NOT NULL DEFAULT '/',
    tls_enabled         BOOLEAN NOT NULL DEFAULT false,
    certificate_id      UUID REFERENCES resources(id),     -- TLS cert resource
    annotations         JSONB NOT NULL DEFAULT '{}',
    upstream_protocol   TEXT NOT NULL DEFAULT 'http',
    status              TEXT NOT NULL DEFAULT 'pending',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(application_id)
);
```

#### autoscalers

```sql
CREATE TABLE autoscalers (
    id                          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    application_id              UUID NOT NULL REFERENCES applications(id) ON DELETE CASCADE,
    enabled                     BOOLEAN NOT NULL DEFAULT false,
    min_replicas                INT NOT NULL DEFAULT 1,
    max_replicas                INT NOT NULL DEFAULT 5,
    target_cpu_percent          INT,     -- e.g. 80
    target_memory_percent       INT,
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at                  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(application_id)
);
```

#### deployments

A record of each deployment attempt.

```sql
CREATE TABLE deployments (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES organizations(id),
    application_id  UUID NOT NULL REFERENCES applications(id) ON DELETE CASCADE,
    workspace_id    UUID NOT NULL REFERENCES workspaces(id),
    status          TEXT NOT NULL DEFAULT 'pending',  -- 'pending', 'in_progress', 'succeeded', 'failed', 'rolled_back'
    trigger_type    TEXT NOT NULL DEFAULT 'manual',   -- 'manual', 'api', 'git_push', 'rollback'
    trigger_value   TEXT,                             -- commit SHA, image tag, etc.
    triggered_by    UUID REFERENCES users(id),
    config_snapshot JSONB,                            -- snapshot of app config at deploy time
    error_message   TEXT,
    started_at      TIMESTAMPTZ,
    finished_at     TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    rollback_of     UUID REFERENCES deployments(id)   -- if this is a rollback
);

CREATE INDEX idx_deployments_app ON deployments(application_id, created_at DESC);
```

#### operations

Generic operation tracking for resources, applications, and managed services.

```sql
CREATE TABLE operations (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES organizations(id),
    target_id       UUID NOT NULL,                    -- resource_id, application_id, etc.
    target_type     TEXT NOT NULL,                     -- 'resource', 'application', 'managed_service'
    action          TEXT NOT NULL,                     -- 'provision', 'upgrade', 'destroy', 'deploy', 'uninstall'
    status          TEXT NOT NULL DEFAULT 'pending',   -- 'pending', 'in_progress', 'succeeded', 'failed'
    details         JSONB,
    error_message   TEXT,
    triggered_by    UUID REFERENCES users(id),
    started_at      TIMESTAMPTZ,
    finished_at     TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_operations_target ON operations(target_id, created_at DESC);
```

#### notification_rules

```sql
CREATE TABLE notification_rules (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL REFERENCES organizations(id),
    workspace_id        UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    event_type          TEXT NOT NULL,   -- 'deployment_succeeded', 'deployment_failed', 'pod_crash', 'pod_oom'
    destination_type    TEXT NOT NULL,   -- 'slack', 'webhook'
    connection_id       UUID REFERENCES connections(id),
    config              JSONB NOT NULL,  -- channel name, webhook URL, etc.
    enabled             BOOLEAN NOT NULL DEFAULT true,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(workspace_id, event_type, destination_type)
);
```

#### jobs

PostgreSQL-backed job queue (used by River or custom implementation).

```sql
CREATE TABLE jobs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES organizations(id),
    type            TEXT NOT NULL,        -- 'provision_resource', 'deploy_application', 'validate_connection', 'destroy_resource'
    status          TEXT NOT NULL DEFAULT 'pending',  -- 'pending', 'running', 'succeeded', 'failed', 'retrying'
    payload         JSONB NOT NULL,
    result          JSONB,
    error_message   TEXT,
    attempts        INT NOT NULL DEFAULT 0,
    max_attempts    INT NOT NULL DEFAULT 3,
    scheduled_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    started_at      TIMESTAMPTZ,
    finished_at     TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_jobs_pending ON jobs(scheduled_at) WHERE status = 'pending';
```

---

## 4. API Design

### 4.1 URL Structure

```
/api/v1/organizations
/api/v1/organizations/{org_id}/members
/api/v1/organizations/{org_id}/connections
/api/v1/organizations/{org_id}/workspaces
/api/v1/organizations/{org_id}/workspaces/{ws_id}/resources
/api/v1/organizations/{org_id}/workspaces/{ws_id}/applications
/api/v1/organizations/{org_id}/workspaces/{ws_id}/applications/{app_id}/containers
/api/v1/organizations/{org_id}/workspaces/{ws_id}/applications/{app_id}/deployments
/api/v1/organizations/{org_id}/workspaces/{ws_id}/notification-rules
```

Tenant scoping comes from the JWT. The `{org_id}` in the URL is validated against the JWT's tenant claim — you can't access an org you're not a member of.

### 4.2 Authentication

Two auth mechanisms:

1. **JWT Bearer tokens** — for the web UI and user API calls
   - `POST /api/v1/auth/register` — create account
   - `POST /api/v1/auth/login` — returns JWT (access + refresh)
   - `POST /api/v1/auth/refresh` — refresh access token
   - Access token: 15 min expiry, contains `{user_id, email, org_memberships}`
   - Refresh token: 7 day expiry, stored hashed in DB

2. **API keys** — for CI/CD and programmatic access
   - Sent via `X-API-Key` header
   - Scoped to a workspace with an access level
   - Looked up by prefix, verified by bcrypt hash

### 4.3 Endpoints

#### Auth

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/auth/register` | Create user account |
| POST | `/api/v1/auth/login` | Login, returns JWT pair |
| POST | `/api/v1/auth/refresh` | Refresh access token |

#### Organizations

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/organizations` | Create organization |
| GET | `/api/v1/organizations` | List user's organizations |
| GET | `/api/v1/organizations/{org_id}` | Get organization |
| PUT | `/api/v1/organizations/{org_id}` | Update organization |

#### Members

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/organizations/{org_id}/members` | List members |
| POST | `/api/v1/organizations/{org_id}/members/invite` | Invite user by email |
| PUT | `/api/v1/organizations/{org_id}/members/{member_id}` | Update role |
| DELETE | `/api/v1/organizations/{org_id}/members/{member_id}` | Remove member |

#### Connections

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/organizations/{org_id}/connections` | Create connection |
| GET | `/api/v1/organizations/{org_id}/connections` | List connections |
| GET | `/api/v1/organizations/{org_id}/connections/{conn_id}` | Get connection (credentials redacted) |
| PUT | `/api/v1/organizations/{org_id}/connections/{conn_id}` | Update connection |
| DELETE | `/api/v1/organizations/{org_id}/connections/{conn_id}` | Delete connection |
| POST | `/api/v1/organizations/{org_id}/connections/{conn_id}/validate` | Validate credentials |

#### Workspaces

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/organizations/{org_id}/workspaces` | Create workspace |
| GET | `/api/v1/organizations/{org_id}/workspaces` | List workspaces |
| GET | `/api/v1/organizations/{org_id}/workspaces/{ws_id}` | Get workspace |
| PUT | `/api/v1/organizations/{org_id}/workspaces/{ws_id}` | Update workspace |
| DELETE | `/api/v1/organizations/{org_id}/workspaces/{ws_id}` | Delete workspace |

#### API Keys

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/organizations/{org_id}/workspaces/{ws_id}/api-keys` | Create API key (returns secret once) |
| GET | `/api/v1/organizations/{org_id}/workspaces/{ws_id}/api-keys` | List API keys (prefix only) |
| DELETE | `/api/v1/organizations/{org_id}/workspaces/{ws_id}/api-keys/{key_id}` | Revoke API key |

#### Resources (Infrastructure)

| Method | Path | Description |
|--------|------|-------------|
| POST | `.../workspaces/{ws_id}/resources` | Create resource |
| GET | `.../workspaces/{ws_id}/resources` | List resources (filterable by type, status) |
| GET | `.../workspaces/{ws_id}/resources/{res_id}` | Get resource with attributes |
| PUT | `.../workspaces/{ws_id}/resources/{res_id}` | Update resource arguments |
| DELETE | `.../workspaces/{ws_id}/resources/{res_id}` | Destroy resource |
| POST | `.../workspaces/{ws_id}/resources/{res_id}/provision` | Trigger provisioning |
| POST | `.../workspaces/{ws_id}/resources/{res_id}/retry` | Retry failed provisioning |
| GET | `.../workspaces/{ws_id}/resources/{res_id}/operations` | List operations history |
| GET | `.../workspaces/{ws_id}/resources/{res_id}/metrics` | Get resource metrics |
| GET | `.../workspaces/{ws_id}/resources/{res_id}/credentials` | Generate temp access credentials |

#### Applications

| Method | Path | Description |
|--------|------|-------------|
| POST | `.../workspaces/{ws_id}/applications` | Create application |
| GET | `.../workspaces/{ws_id}/applications` | List applications |
| GET | `.../workspaces/{ws_id}/applications/{app_id}` | Get application with full config |
| PUT | `.../workspaces/{ws_id}/applications/{app_id}` | Update application |
| DELETE | `.../workspaces/{ws_id}/applications/{app_id}` | Remove application |
| GET | `.../workspaces/{ws_id}/applications/{app_id}/status` | Get runtime status (pods, events) |

#### Containers

| Method | Path | Description |
|--------|------|-------------|
| POST | `.../applications/{app_id}/containers` | Add container |
| GET | `.../applications/{app_id}/containers` | List containers |
| PUT | `.../applications/{app_id}/containers/{ctr_id}` | Update container spec |
| DELETE | `.../applications/{app_id}/containers/{ctr_id}` | Remove container |
| PUT | `.../containers/{ctr_id}/env-vars` | Set environment variables (bulk upsert) |
| GET | `.../containers/{ctr_id}/env-vars` | List environment variables |
| PUT | `.../containers/{ctr_id}/ports` | Set ports (bulk upsert) |
| PUT | `.../containers/{ctr_id}/probes` | Set probes |
| PUT | `.../containers/{ctr_id}/volumes` | Set volumes |
| PUT | `.../containers/{ctr_id}/resources` | Set CPU/memory requests and limits |

#### Ingress

| Method | Path | Description |
|--------|------|-------------|
| PUT | `.../applications/{app_id}/ingress` | Create/update ingress |
| GET | `.../applications/{app_id}/ingress` | Get ingress config |
| DELETE | `.../applications/{app_id}/ingress` | Remove ingress |

#### Autoscaling

| Method | Path | Description |
|--------|------|-------------|
| PUT | `.../applications/{app_id}/autoscaler` | Create/update autoscaler config |
| GET | `.../applications/{app_id}/autoscaler` | Get autoscaler config |

#### Deployments

| Method | Path | Description |
|--------|------|-------------|
| POST | `.../applications/{app_id}/deployments` | Trigger deployment |
| GET | `.../applications/{app_id}/deployments` | List deployments |
| GET | `.../applications/{app_id}/deployments/{dep_id}` | Get deployment details |
| POST | `.../applications/{app_id}/deployments/{dep_id}/rollback` | Rollback to this deployment |
| GET | `.../applications/{app_id}/deployments/{dep_id}/diff` | Config diff vs previous |

#### Notification Rules

| Method | Path | Description |
|--------|------|-------------|
| POST | `.../workspaces/{ws_id}/notification-rules` | Create rule |
| GET | `.../workspaces/{ws_id}/notification-rules` | List rules |
| PUT | `.../workspaces/{ws_id}/notification-rules/{rule_id}` | Update rule |
| DELETE | `.../workspaces/{ws_id}/notification-rules/{rule_id}` | Delete rule |

#### External Deploy API (API Key Auth)

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/deploy` | Deploy application by name + image tag |
| GET | `/api/v1/deploy/{deployment_id}` | Get deployment status |

This is the endpoint CI/CD pipelines call. Authenticated via `X-API-Key` header, scoped to a workspace.

```json
POST /api/v1/deploy
{
    "application": "my-app",
    "containers": [
        { "name": "main", "image_tag": "v1.2.3" }
    ]
}
```

#### Health & Meta

| Method | Path | Description |
|--------|------|-------------|
| GET | `/healthz` | Health check |
| GET | `/readyz` | Readiness check (DB connected) |
| GET | `/api/v1/version` | Server version |

---

## 5. Cloud Provider Interface

```go
// provider.go

type Provider interface {
    // Identity
    ValidateCredentials(ctx context.Context) error
    GetAccountInfo(ctx context.Context) (*AccountInfo, error)

    // Infrastructure provisioning (delegates to Terraform/OpenTofu)
    GetResourceTypes() []ResourceTypeDefinition
    GetDefaultArguments(resourceType string) map[string]interface{}
    GenerateTerraform(resource *Resource, dependencies []*Resource) ([]byte, error)

    // Compute — Kubernetes
    GetKubeConfig(ctx context.Context, clusterID string) (*rest.Config, error)
    ListClusters(ctx context.Context) ([]ClusterInfo, error)

    // Container registries
    ListRepositories(ctx context.Context, connectionConfig map[string]string) ([]Repository, error)
    ListImageTags(ctx context.Context, connectionConfig map[string]string, repo string) ([]ImageTag, error)

    // Metrics
    GetResourceMetrics(ctx context.Context, providerID string, metricName string, window TimeRange) ([]DataPoint, error)

    // Cost
    GetCostBreakdown(ctx context.Context, filters CostFilter) (*CostReport, error)

    // Temporary credentials
    GenerateTemporaryCredentials(ctx context.Context, resource *Resource) (*TemporaryCredentials, error)
}

type ResourceTypeDefinition struct {
    Type        string                  // "database", "cache", "network", etc.
    DisplayName string
    Arguments   []ArgumentDefinition    // what the user configures
    Attributes  []AttributeDefinition   // what comes back after provisioning
}

type ArgumentDefinition struct {
    Name         string
    Type         string   // "string", "int", "bool", "enum"
    Required     bool
    Default      interface{}
    Options      []string  // for enums
    Description  string
    Group        string    // UI grouping
}
```

The `GenerateTerraform` method produces HCL that the provisioning engine writes to disk and applies via `tofu apply`. The provider knows how to map Kapstan resource types to cloud-specific Terraform modules.

AWS is implemented first. GCP and Azure follow the same interface.

---

## 6. Background Jobs

### 6.1 Job Queue

PostgreSQL-backed using River (https://github.com/riverqueue/river). Jobs are transactional with application data — when you create a resource row, you enqueue the provisioning job in the same transaction. No two-phase commit problems.

### 6.2 Job Types

| Job Type | Trigger | What it does |
|----------|---------|--------------|
| `provision_resource` | User creates/updates resource | Generates Terraform, runs `tofu apply`, stores outputs |
| `destroy_resource` | User deletes resource | Runs `tofu destroy`, cleans up |
| `deploy_application` | User triggers deploy | Generates Helm values, runs Helm install/upgrade |
| `uninstall_application` | User removes app | Runs Helm uninstall |
| `validate_connection` | User creates connection or periodic | Tests cloud credentials are still valid |
| `send_notification` | Deployment events | Sends Slack/webhook notification |
| `cost_sync` | Periodic (daily) | Fetches cost data from cloud provider APIs |
| `connection_check` | Periodic (hourly) | Validates all connections are still healthy |

### 6.3 Periodic Jobs

Scheduled via a simple in-process cron (like `robfig/cron`):

- **Connection validation**: hourly — checks all connections are still valid
- **Cost sync**: daily — pulls cost data from cloud cost APIs

No external scheduler. No Temporal. No separate process.

---

## 7. Frontend

### 7.1 Stack

- React 18 + TypeScript
- Vite for build tooling
- TanStack Query for server state
- TanStack Router for type-safe routing
- Tailwind CSS for styling (or keep existing component library if preferred)
- OpenAPI-generated TypeScript client (via `openapi-typescript-codegen` or similar)

### 7.2 Deployment

The frontend is a standalone service, separate from the Go backend. In production, the React app is built into its own container image and served via nginx (or any static file server). The backend is a pure REST API server with no static file serving.

In development, Vite's dev server runs on port 5173 and proxies API requests to the Go backend on port 8080.

### 7.3 Pages

| Page | Route | Description |
|------|-------|-------------|
| Login/Register | `/auth/*` | Authentication |
| Dashboard | `/orgs/{org}/` | Overview of workspaces |
| Workspace | `/orgs/{org}/workspaces/{ws}` | Resources + applications in this environment |
| Resource detail | `.../resources/{id}` | Resource config, status, metrics, operations |
| Application detail | `.../applications/{id}` | Containers, config, deployments, status |
| Deployment detail | `.../deployments/{id}` | Deploy status, logs, config diff |
| Connections | `/orgs/{org}/connections` | Manage cloud credentials |
| Settings | `/orgs/{org}/settings` | Members, roles, API keys, notification rules |

---

## 8. Build Phases

Each phase produces a working, testable increment.

---

### Phase 0 — Scaffolding (1 week)

**Goal:** `kapstan server` starts and serves health endpoints. React dev server runs separately.

**Deliverables:**
- Go module initialization
- CLI setup with cobra: `kapstan server` command
- Echo HTTP router with `/healthz`, `/readyz`
- PostgreSQL connection via sqlx with golang-migrate
- Configuration via environment variables (DATABASE_URL, PORT, KAPSTAN_ENCRYPTION_KEY)
- Structured logging with slog
- React app skeleton with Vite + TypeScript (standalone service in `web/`)
- Dockerfile for Go backend (multi-stage: go build → alpine)
- Dockerfile for frontend (multi-stage: node build → nginx)
- Makefile with targets: `build`, `dev`, `test`, `lint`, `migrate`
- GitHub Actions CI: lint + test + build

**First migration:** just the `schema_migrations` table.

---

### Phase 1 — Auth + Multi-Tenancy (2 weeks)

**Goal:** Users can register, log in, create organizations, invite members.

**Deliverables:**
- Database migrations: `users`, `organizations`, `organization_members`, `api_keys`
- Auth endpoints: register, login, refresh
- JWT middleware (access token validation, tenant context extraction)
- RBAC middleware (role checks per endpoint)
- Organization CRUD endpoints
- Member management (invite, update role, remove)
- API key creation and authentication middleware
- Frontend: login page, org creation, member management
- Password hashing with bcrypt
- Tests for auth flow, RBAC, multi-tenant isolation

**Key test:** User A in Org1 cannot see Org2's data, even by crafting URLs.

---

### Phase 2 — Connections (1 week)

**Goal:** Users can register AWS credentials and validate them.

**Deliverables:**
- Database migration: `connections`
- AES-256-GCM encryption module for credentials at rest
- Connection CRUD endpoints
- Provider interface definition (`provider.go`)
- AWS provider: `ValidateCredentials` implementation (STS GetCallerIdentity)
- Connection validation endpoint
- Background job: `validate_connection`
- Job runner setup (River + PostgreSQL)
- Frontend: connections page, add AWS connection form, validation status
- Tests for encryption round-trip, credential validation

---

### Phase 3 — Workspaces + Infrastructure (3 weeks)

**Goal:** Users can create workspaces, define infrastructure resources, and provision them via Terraform.

**Week 1 — Workspaces + resource definitions:**
- Database migrations: `workspaces`, `workspace_members`, `resources`, `resource_dependencies`, `operations`
- Workspace CRUD endpoints
- Resource CRUD endpoints (create, list, get, update, delete)
- AWS provider: `GetResourceTypes`, `GetDefaultArguments`
- Resource type definitions for: network (VPC), kubernetes (EKS), database (RDS), node_group, cache (ElastiCache)
- Frontend: workspace page, resource list, create resource form with dynamic arguments

**Week 2 — Provisioning engine:**
- AWS provider: `GenerateTerraform` for each resource type
- Provisioning job: write HCL to temp dir, run `tofu init && tofu apply`, parse outputs
- Resource status tracking (pending → provisioning → active/failed)
- Operation history per resource
- Dependency resolution: provision in topological order
- Frontend: resource status indicators, operation history panel

**Week 3 — Resource management:**
- Resource upgrade (change arguments, re-apply)
- Resource destroy
- Resource metrics (AWS CloudWatch via provider)
- Temporary credential generation (for database access, etc.)
- Import existing cloud resources
- Frontend: resource detail page with metrics, credential generation, operations

---

### Phase 4 — Applications (2 weeks)

**Goal:** Users can define applications with containers, environment variables, ports, probes, and autoscaling.

**Week 1 — App + container definitions:**
- Database migrations: `applications`, `containers`, `container_ports`, `container_env_vars`, `container_probes`, `container_volumes`, `ingresses`, `autoscalers`, `workspace_env_vars`
- Application CRUD endpoints
- Container CRUD endpoints with nested resources (ports, env vars, probes, volumes)
- Ingress and autoscaler endpoints
- Workspace-level environment variable endpoints
- AWS provider: `ListRepositories`, `ListImageTags` for ECR
- Frontend: application list, create app form, container editor, env var management

**Week 2 — App configuration experience:**
- Config diffing between current and last deployed
- Bulk env var upload/download
- Image tag picker (from container registry)
- Application cloning
- Frontend: full application detail page with all config tabs

---

### Phase 5 — Deployment (2 weeks)

**Goal:** Users can deploy applications to Kubernetes and track deployment status.

**Week 1 — Deployment engine:**
- Database migration: `deployments`
- Helm value generation from app config (containers, env vars, ports, probes, volumes, ingress, autoscaler)
- Deploy job: Helm install/upgrade via Go SDK
- Deployment status tracking
- K8s status polling (pod status, events)
- AWS provider: `GetKubeConfig` for EKS clusters
- Rollback support (redeploy previous config snapshot)
- Frontend: deploy button, deployment list, deployment status page

**Week 2 — CD + external deploy API:**
- External deploy API (`POST /api/v1/deploy`) with API key auth
- CI/CD configuration generator (GitHub Actions YAML snippet)
- Deployment config diff view
- Notification rules + send_notification job (Slack webhook, generic webhook)
- Frontend: deployment diff, CD settings, notification rule management

---

### Phase 6 — Cost + Monitoring (2 weeks)

**Goal:** Cost tracking per resource and application. Basic monitoring dashboards.

**Deliverables:**
- AWS provider: `GetCostBreakdown` via Cost Explorer API
- Cost sync periodic job
- Cost breakdown endpoints (by workspace, by resource, by application)
- Resource metrics endpoints (CloudWatch)
- Application metrics (pod CPU/memory via K8s metrics API)
- Frontend: cost dashboard, resource metrics charts, application metrics

---

### Phase 7 — Polish + GCP Provider (2 weeks)

**Goal:** Second cloud provider, workspace cloning, remaining quality-of-life features.

**Deliverables:**
- GCP provider implementation (same interface)
- Workspace cloning (deep copy resources + apps to new workspace)
- Managed services support (cluster add-ons like cert-manager, external-dns)
- Audit log (who did what, when)
- Helm chart for self-hosted installation
- Documentation site
- Frontend polish, error handling, loading states

---

## 9. Development Commands

```bash
# Start development (Go backend + Vite frontend with hot reload)
make dev

# Run all tests
make test

# Run a single test
go test -v -run TestResourceProvisioning ./internal/services/infrastructure/

# Run linter
make lint

# Run database migrations
make migrate

# Build Go backend binary
make build

# Build Docker image
make docker

# Full CI check (lint + test + build)
make ci
```

---

## 10. Configuration

All configuration via environment variables. No config files required.

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DATABASE_URL` | Yes | — | PostgreSQL connection string |
| `KAPSTAN_ENCRYPTION_KEY` | Yes | — | 32-byte hex key for AES-256-GCM |
| `PORT` | No | `8080` | HTTP listen port |
| `JWT_SECRET` | No | derived from encryption key | JWT signing key |
| `REDIS_URL` | No | — | Redis URL (optional caching) |
| `LOG_LEVEL` | No | `info` | Logging level |
| `LOG_FORMAT` | No | `json` | `json` or `text` |

---

## 11. What's Explicitly NOT in Scope

These are intentional exclusions for the initial open-source release:

- **gRPC / protobuf** — REST only
- **Temporal** — PostgreSQL job queue instead
- **NATS** — in-process function calls
- **Elasticsearch** — not needed without Temporal
- **Vault** — encryption key from env var, optional KMS later
- **OPA** — RBAC in application code
- **Auth0** — built-in auth, optional OIDC later
- **Multiple observability backends** — just Prometheus `/metrics` endpoint
- **Ray/KubeRay support** — niche, add later if demanded
- **KEDA autoscaling** — HPA only for now
- **Architecture graph visualization** — nice-to-have, not MVP
- **Azure provider** — Phase 7+ (after AWS and GCP prove the interface)
