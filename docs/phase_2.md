# Phase 2 — Connections

Build Phase 2 from `docs/SPEC.md` — cloud connections and credential management.

## Prerequisites

Phase 1 is complete. Auth, multi-tenancy, organizations, and RBAC are working.

## Deliverables

### Database Migration

Create migration for the `connections` table (see `docs/SPEC.md` Section 3.2):

- id, tenant_id, name (unique per tenant), category, status, last_validated, credentials (BYTEA — encrypted), config (JSONB — non-secret), created_by, timestamps

### Encryption Module (`internal/platform/crypto/`)

1. AES-256-GCM encryption and decryption functions
2. Key derived from `KAPSTAN_ENCRYPTION_KEY` environment variable (32-byte hex string)
3. Each encrypted value gets a unique random nonce (prepended to ciphertext)
4. Functions: `Encrypt(plaintext []byte) ([]byte, error)` and `Decrypt(ciphertext []byte) ([]byte, error)`
5. Tests for round-trip encryption, wrong key detection, tampered ciphertext detection

### Provider Interface (`internal/provider/provider.go`)

Define the cloud provider interface as specified in `docs/SPEC.md` Section 5. For this phase, only implement `ValidateCredentials` and `GetAccountInfo` on the AWS provider. The rest return "not implemented" errors — they'll be built in later phases.

### AWS Provider (`internal/provider/aws/`)

1. Constructor: takes decrypted credentials (access key ID, secret access key, optional role ARN, region)
2. `ValidateCredentials` — calls AWS STS `GetCallerIdentity`. Returns nil if credentials work, error if not.
3. `GetAccountInfo` — returns account ID, ARN from the STS response
4. If `role_arn` is provided in credentials, assume the role first via STS `AssumeRole`

### Provider Factory

A function that takes a `Connection` (with decrypted credentials) and returns the appropriate `Provider` based on the connection's `category`. For now, only `aws` is supported; others return an error.

### Background Job System (`internal/jobs/`)

1. Set up River with PostgreSQL as the backend (River uses pgx internally — this is fine since it's an internal dependency of River, not our direct DB access which uses sqlx)
2. River migration (add River's required tables)
3. Job runner that starts with the server and processes jobs
4. First job type: `validate_connection` — takes a connection ID, decrypts credentials, calls `provider.ValidateCredentials()`, updates the connection's `status` and `last_validated` fields
5. Periodic scheduler using River's periodic jobs — schedule `validate_connection` for all connections every hour

### Connection Service Logic (`internal/services/connection/`)

1. `Create` — validate input, encrypt credentials, store connection, enqueue `validate_connection` job
2. `Get` — return connection with credentials redacted (just show category, status, config, last_validated)
3. `List` — all connections for the tenant, credentials redacted
4. `Update` — update name, config, or credentials (re-encrypt if changed), enqueue validation job
5. `Delete` — delete the connection
6. `Validate` — manually trigger validation (enqueue job, return immediately)
7. `GetDecrypted` — internal only (not exposed via API), returns decrypted credentials. Used by provider factory and other service packages.

### Repository Layer (`internal/repositories/connection.go`)

Write sqlx repository functions for connection CRUD. Store credentials as `BYTEA` (encrypted bytes). Store config as `JSONB`.

### Controllers

All under `/api/v1/organizations/{org_id}/connections`, JWT auth + tenant middleware. Use Echo route groups.

- `POST /` — create connection. Encrypt credentials before storage. Return connection without credentials.
- `GET /` — list connections (credentials never returned)
- `GET /{conn_id}` — get connection detail (credentials never returned)
- `PUT /{conn_id}` — update connection
- `DELETE /{conn_id}` — delete connection (admin+ only)
- `POST /{conn_id}/validate` — trigger validation, returns current status

### Frontend

1. **Connections page** (`/orgs/{org_id}/connections`) — list all connections with name, category, status, last validated time
2. **Create connection modal/page** — form that adapts based on selected category (show different fields for AWS vs GitHub vs Slack). Only AWS validation works for now.
3. **Connection detail** — shows status, config (non-secret), last validated time, validate button
4. **Status indicators** — green for valid, red for invalid, yellow for pending/validating
5. Add connections link to the org sidebar/navigation

### Tests

1. **Encryption round-trip** — encrypt → decrypt returns original. Different plaintexts produce different ciphertexts (random nonce).
2. **Connection CRUD** — create, list, get, update, delete. Verify credentials are never returned in API responses.
3. **AWS validation** — mock STS client. Valid credentials → status becomes "valid". Invalid → "invalid".
4. **Tenant isolation** — connection created in Org A is not visible from Org B.
5. **Job execution** — create connection → job is enqueued → job runs → status is updated.
6. **Credential encryption in DB** — read raw row from DB, verify the credentials column is not plaintext.

## Constraints

- Credentials must NEVER appear in API responses, logs, or error messages
- Credentials must be encrypted at rest — the raw bytes in PostgreSQL must be undecipherable without the key
- The provider interface must be defined completely (all methods), even though most return "not implemented" for now
- Do not import AWS SDK in any package outside `internal/provider/aws/`
- The job system must be transactional — connection creation and job enqueue should be coordinated

## Verification

```bash
make test

# Create an AWS connection
curl -X POST http://localhost:8080/api/v1/organizations/{org_id}/connections \
  -H 'Authorization: Bearer <token>' \
  -H 'Content-Type: application/json' \
  -d '{"name":"my-aws","category":"aws","credentials":{"access_key_id":"...","secret_access_key":"...","region":"us-east-1"}}'
# → returns connection with status "pending", no credentials in response

# After job runs, status updates to "valid" or "invalid"
curl http://localhost:8080/api/v1/organizations/{org_id}/connections/{conn_id} \
  -H 'Authorization: Bearer <token>'
# → status: "valid", last_validated: "2025-..."
```
