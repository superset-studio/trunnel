# Phase 5 — Deployment

Build Phase 5 from `docs/SPEC.md` — deploy applications to Kubernetes, track deployments, enable CI/CD.

## Prerequisites

Phase 4 is complete. Applications, containers, env vars, ports, probes, volumes, ingress, and autoscaler configuration are all working. Config snapshots and diffing work.

## Deliverables

---

## Phase 5a — Deployment Engine (Week 1)

### Database Migration

Create migration for `deployments` table (see `docs/SPEC.md` Section 3.2):

- id, tenant_id, application_id, workspace_id, status (pending/in_progress/succeeded/failed/rolled_back), trigger_type (manual/api/git_push/rollback), trigger_value, triggered_by, config_snapshot (JSONB), error_message, started_at, finished_at, created_at, rollback_of (self-referencing FK)
- Index on (application_id, created_at DESC)

### Embedded Helm Chart

Create `internal/services/deployment/chart/` — a generic Helm chart embedded via `go:embed`:

```
chart/
├── Chart.yaml
├── values.yaml
└── templates/
    ├── deployment.yaml       # (or cronjob.yaml based on .Values.type)
    ├── service.yaml
    ├── ingress.yaml
    ├── hpa.yaml
    ├── secrets.yaml
    ├── configmap.yaml
    ├── pvc.yaml
    └── _helpers.tpl
```

The chart handles both Deployment and CronJob types via conditional templating.

### Helm Value Generation (`internal/services/deployment/`)

Build a function that takes an application's full config snapshot and generates Helm values:

1. Container specs → image, tag, pull policy, command, args, resource requests/limits
2. Ports → containerPort definitions, Service port mappings
3. Environment variables → plain values as ConfigMap entries, secrets as K8s Secret entries via secretKeyRef
4. Probes → liveness, readiness, startup probe specs
5. Volumes → PVC definitions and volume mounts
6. Ingress → ingress resource with hostname, path, TLS, annotations
7. Autoscaler → HPA spec with min/max replicas and target utilization
8. Replicas, namespace, labels, annotations

### Deploy Job (`internal/jobs/`)

New job type: `deploy_application`

1. Load application and full config
2. Create config snapshot, store in deployment record
3. Get kubeconfig for the target cluster:
   - Call `provider.GetKubeConfig(ctx, clusterProviderID)` using the workspace connection
   - Or use the workspace's `cluster_config` JSONB if it contains a direct kubeconfig
4. Generate Helm values from config
5. Execute Helm install/upgrade via the Helm Go SDK (`helm.sh/helm/v3`):
   - Release name: `kapstan-{app_name}`
   - Namespace: from application config (create if not exists)
   - Install if first deploy, upgrade if release exists
   - Wait for rollout (configurable timeout, default 5 min)
6. Poll Kubernetes for pod status until all pods are ready or timeout
7. Update deployment status (succeeded/failed) and application status (running/failed)
8. Create operation record

### Uninstall Job

New job type: `uninstall_application`
- Helm uninstall the release, update application status to "stopped", create operation record

### Rollback

1. Load the target deployment's `config_snapshot`
2. Create new deployment record with `trigger_type: "rollback"` and `rollback_of: {target_deployment_id}`
3. Generate Helm values from the old snapshot
4. Execute Helm upgrade with those values

### AWS Provider — GetKubeConfig

Implement `GetKubeConfig(ctx, clusterID)`:
1. Call EKS `DescribeCluster` to get cluster endpoint and CA
2. Generate a kubeconfig using AWS IAM authenticator token
3. Return a `*rest.Config` usable by the Helm SDK and K8s client

### Deployment Service Logic (`internal/services/deployment/`)

1. `Deploy(appID, triggerType, triggerValue, triggeredBy)` — snapshot config, create deployment, enqueue job
2. `Rollback(appID, targetDeploymentID, triggeredBy)` — create rollback deployment, enqueue job
3. `List(appID)` — paginated, newest first
4. `Get(deploymentID)` — full details including config snapshot
5. `GetDiff(deploymentID)` — diff vs previous deployment's config

### Controllers

#### Deployment endpoints (`.../applications/{app_id}/deployments`)
- `POST /` — trigger deployment
- `GET /` — list deployments (paginated)
- `GET /{dep_id}` — deployment details
- `POST /{dep_id}/rollback` — rollback to this deployment
- `GET /{dep_id}/diff` — config diff vs previous

#### Application status
- `GET .../applications/{app_id}/status` — runtime status from K8s: pod count, events, resource usage

### Frontend

1. **Deploy button** on application detail page with confirmation
2. **Deployment list** — table with status badge, trigger type, triggered by, time, duration, rollback action
3. **Deployment detail** — status, pod statuses, config snapshot, diff viewer
4. **Rollback confirmation modal**
5. **Application status panel** — pod count, pod names with status, events. Auto-refresh every 10s.
6. **Deploy in progress indicator** — spinner on app card and detail page

### Tests

1. Helm value generation — snapshot test: given app config, verify generated values match expected structure
2. Deploy job — mock Helm SDK, verify snapshot stored, Helm called correctly, status updated
3. Rollback — old config snapshot used, new deployment record created
4. Diff — deploy twice with changes, verify diff shows the changes
5. Uninstall — Helm uninstall called, status updated
6. Deployment listing — pagination, ordering

---

## Phase 5b — CD + External API + Notifications (Week 2)

### External Deploy API

API key auth (not JWT). Uses Echo middleware.

- `POST /api/v1/deploy` — deploy by application name + image tags:
  ```json
  {
    "application": "my-app",
    "containers": [
      { "name": "main", "image_tag": "v1.2.3" }
    ]
  }
  ```
  Look up app by name in the API key's workspace, update container image tags, trigger deployment with `trigger_type: "api"`.

- `GET /api/v1/deploy/{deployment_id}` — poll deployment status

### CI/CD Config Generator

Read-only endpoints that generate pipeline config snippets:
- `GET .../applications/{app_id}/cd/github-actions` — GitHub Actions YAML
- `GET .../applications/{app_id}/cd/generic` — curl-based deploy script

### Notification System

#### Database Migration
Create `notification_rules` table (see `docs/SPEC.md` Section 3.2)

#### Notification Service Logic (`internal/services/notification/`)
1. `CreateRule` / `ListRules` / `UpdateRule` / `DeleteRule`
2. `Dispatch(event)` — called internally on events, looks up matching rules, enqueues send jobs

#### Supported Events
- `deployment_succeeded`
- `deployment_failed`

#### Supported Destinations
- `slack` — message via webhook URL
- `webhook` — JSON POST to configured URL

#### Notification Job
New job type: `send_notification` — format message, send HTTP request, log outcome

#### Integration
After deploy job completes, call `notification.Dispatch()` with the appropriate event.

### Controllers (`.../workspaces/{ws_id}/notification-rules`)
- `POST /`, `GET /`, `PUT /{rule_id}`, `DELETE /{rule_id}`

### Frontend Additions
1. CD settings on application detail — toggle, API key instructions, generated CI/CD config
2. Notification rules page — list rules, create rule form
3. Deployment trigger info — shows trigger type and value

### Tests
1. External deploy API — create API key, call deploy, verify deployment created
2. API key scoping — key for workspace A cannot deploy to workspace B
3. Notification dispatch — deploy succeeds → Slack webhook called (mock HTTP)
4. CD config generation — verify generated YAML is valid

## Constraints

- Embedded Helm chart must be self-contained (no external chart repo dependencies)
- Helm operations must have a timeout (default 5 min)
- Deploy API must be idempotent
- Notification dispatch must not block the deploy job (separate job)
- Config snapshots are immutable once created
- External deploy API must validate that all container names exist in the application
