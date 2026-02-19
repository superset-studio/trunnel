# Phase 4 — Applications

Build Phase 4 from `docs/SPEC.md` — application definitions, container specs, and configuration management.

## Prerequisites

Phase 3 is complete. Workspaces, resources, infrastructure provisioning, and the job system are working. Kubernetes cluster resources can be provisioned.

## Deliverables

---

## Phase 4a — App + Container Definitions (Week 1)

### Database Migrations

Create migrations for (see `docs/SPEC.md` Section 3.2):

1. `applications` — id, tenant_id, workspace_id, cluster_id (FK to resources), name (unique per workspace), display_name, namespace, type (deployment/cronjob/chart/custom_chart), status, replicas, cd_enabled, imported, created_by, timestamps
2. `containers` — id, application_id, name (unique per app), type (main/sidecar/init), image_uri, image_tag, connection_id (registry creds), pull_policy, command (TEXT[]), args (TEXT[]), cpu_request, cpu_limit, memory_request, memory_limit, timestamps
3. `container_ports` — id, container_id, name (unique per container), container_port, protocol, is_public, timestamps
4. `container_env_vars` — id, container_id, name (unique per container), value, secret_value (BYTEA encrypted), type (plain/secret), timestamps
5. `container_probes` — id, container_id, probe_type (unique per container: liveness/readiness/startup), mechanism (http/tcp/exec), path, port, command, timing params, timestamps
6. `container_volumes` — id, container_id, name (unique per container), mount_path, size, storage_class, read_only, timestamps
7. `workspace_env_vars` — id, workspace_id, name (unique per workspace), value, secret_value (BYTEA encrypted), type, timestamps
8. `ingresses` — id, application_id (unique), hostname, path, tls_enabled, certificate_id (FK to resources), annotations (JSONB), upstream_protocol, status, timestamps
9. `autoscalers` — id, application_id (unique), enabled, min_replicas, max_replicas, target_cpu_percent, target_memory_percent, timestamps

### Repository Layer

Write sqlx repository functions for all CRUD on the above tables. Key patterns:
- Get application with all nested data (multiple queries per controller is fine — avoid complex joins)
- Bulk upsert for env vars, ports, probes (replace all for a container in one call using a transaction)
- List applications with status and container count

### Application Service Logic (`internal/services/application/`)

1. `Create` — validate name uniqueness per workspace, validate cluster_id is an active kubernetes resource in this workspace, set status to "pending"
2. `Get` — full application config (all containers with their ports, env vars, probes, volumes, plus ingress and autoscaler)
3. `List` — applications in a workspace with summary info
4. `Update` — update display_name, namespace, replicas, type
5. `Delete` — enqueue uninstall job (Phase 5), set status to "deleting"
6. `Clone` — deep copy an application within the same workspace or to another workspace (new name required)

### Container Service Logic (within `internal/services/application/`)

1. `AddContainer` — validate name uniqueness per app, validate connection_id if provided
2. `UpdateContainer` — update image, tag, resources, command, args
3. `RemoveContainer` — cannot remove the last main container
4. `SetEnvVars` — bulk upsert. Encrypt secret values. Merge with workspace_env_vars at read time.
5. `GetEnvVars` — return merged env vars (container-level + workspace-level). Secret values returned as `"********"` unless `reveal=true` param is passed (admin+ only).
6. `SetPorts` / `SetProbes` / `SetVolumes` / `SetResources` — replace all for a container

### AWS Provider — Container Registry

Implement for ECR:
1. `ListRepositories` — list ECR repositories for the connection's AWS account
2. `ListImageTags` — list tags for a specific ECR repository, sorted by push date descending

### Controllers (Echo route groups)

#### Application endpoints (`.../workspaces/{ws_id}/applications`)
- `POST /`, `GET /`, `GET /{app_id}`, `PUT /{app_id}`, `DELETE /{app_id}`, `POST /{app_id}/clone`

#### Container endpoints (`.../applications/{app_id}/containers`)
- `POST /`, `GET /`, `PUT /{ctr_id}`, `DELETE /{ctr_id}`
- `PUT /{ctr_id}/env-vars`, `GET /{ctr_id}/env-vars`
- `PUT /{ctr_id}/ports`, `PUT /{ctr_id}/probes`, `PUT /{ctr_id}/volumes`, `PUT /{ctr_id}/resources`

#### Ingress endpoints (`.../applications/{app_id}/ingress`)
- `PUT /`, `GET /`, `DELETE /`

#### Autoscaler endpoints (`.../applications/{app_id}/autoscaler`)
- `PUT /`, `GET /`

#### Workspace env var endpoints (`.../workspaces/{ws_id}/env-vars`)
- `POST /`, `GET /`, `PUT /{var_id}`, `DELETE /{var_id}`

#### Registry endpoints (`.../connections/{conn_id}/registries`)
- `GET /repositories`, `GET /repositories/{repo}/tags`

### Frontend

1. **Application list** on workspace page — table with name, type, status, cluster, last deployed time
2. **Create application modal** — name, display name, namespace, type selector, cluster selector
3. **Application detail page** with tabs: Containers, Environment, Networking, Scaling, Health, Storage
4. **Container editor** — image URI + tag picker (fetches from registry), command, args, resource limits
5. **Image tag picker** — dropdown listing tags from container registry, sorted by most recent

---

## Phase 4b — Configuration Experience (Week 2)

### Config Snapshot

1. `SnapshotConfig(appID) (JSON, error)` — serializes full application config into a single JSON document
2. `DiffConfigs(old, new JSON) ([]Change, error)` — compares two config snapshots, returns list of changes

### Bulk Env Var Upload/Download

- `POST .../containers/{ctr_id}/env-vars/upload` — upload a `.env` file, parse key=value pairs, bulk upsert
- `GET .../containers/{ctr_id}/env-vars/download` — download as `.env` file (secrets excluded unless admin)

### Application Cloning

Deep copy: application row, all containers, ports, env vars (re-encrypted), probes, volumes, ingress, autoscaler. New IDs and name. Status set to "pending".

### Frontend Additions

1. Config diff viewer — side-by-side diff of two config snapshots
2. Env var file upload — drag and drop `.env` file
3. Clone application button — modal with new name, optional new workspace target

### Tests

1. Application CRUD with tenant isolation
2. Container CRUD — add, update, remove. Cannot remove last main container.
3. Env var encryption — secret env vars encrypted in DB, masked in API response, revealable by admin
4. Port uniqueness — duplicate port names per container → error
5. Config snapshot — snapshot captures all config, diff correctly identifies changes
6. Clone — cloned app has all config but new IDs and name
7. Registry listing — mock ECR, verify repositories and tags returned

## Constraints

- Secret env vars must be encrypted at rest (same AES-256-GCM as connection credentials)
- Secret env var values must be masked in API responses by default
- Container/application names must be unique within their parent
- Config snapshot function must be deterministic (same config → same JSON)
- Do not implement deployment execution yet — that's Phase 5. Applications are just definitions at this stage.
