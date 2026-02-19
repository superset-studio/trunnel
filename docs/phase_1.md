# Phase 1 — Auth + Multi-Tenancy

Build Phase 1 from `docs/SPEC.md` — authentication and multi-tenancy.

## Prerequisites

Phase 0 is complete. The project scaffolding, Echo HTTP server, sqlx database connection, and migration system are in place.

## Deliverables

### Database Migrations

Create migrations for these tables (see `docs/SPEC.md` Section 3.2 for full schemas):

1. `users` — id, email (unique), password_hash, name, avatar_url, email_verified, timestamps
2. `organizations` — id, name (unique slug), display_name, logo_url, timestamps
3. `organization_members` — id, organization_id, user_id, role (owner/admin/member/viewer), invited_by, invited_at, accepted_at, timestamps. Unique on (organization_id, user_id)
4. `api_keys` — id, tenant_id, workspace_id (nullable for now — workspaces don't exist yet, add FK later), name, key_prefix, key_hash (unique), access_level, created_by, created_at, last_used_at
5. `refresh_tokens` — id, user_id, token_hash, expires_at, created_at

### Repository Layer (`internal/repositories/`)

Write repository functions using sqlx for all CRUD operations on the above tables. Use `sqlx.Get`, `sqlx.Select`, `sqlx.NamedExec` for queries. Define Go structs with `db:"column_name"` tags for scanning.

### Auth System (`internal/platform/auth/`)

1. Password hashing with bcrypt (cost 12)
2. JWT generation and validation:
   - Access token: 15 min expiry, contains `user_id`, `email`, `org_memberships` (list of `{org_id, role}`)
   - Refresh token: 7 day expiry, stored as bcrypt hash in `refresh_tokens` table
   - Sign with HMAC-SHA256 using `JWT_SECRET` env var (default: derived from `KAPSTAN_ENCRYPTION_KEY`)
3. Refresh token rotation — issuing a new refresh token invalidates the old one

### Middleware (`internal/controllers/middleware/`)

1. **Auth middleware** — extracts and validates JWT from `Authorization: Bearer <token>` header. Sets user context (user_id, email, org memberships) on Echo context.
2. **Tenant middleware** — extracts `org_id` from the URL path, validates the user is a member of that org, sets tenant context on Echo context. Returns 403 if not a member.
3. **RBAC middleware** — configurable per-route. Checks the user's role in the current org meets the minimum required level. Role hierarchy: owner > admin > member > viewer.
4. **API key middleware** — alternative to JWT auth for the external API. Extracts key from `X-API-Key` header, looks up by prefix, verifies bcrypt hash, sets tenant/workspace context. Updates `last_used_at`.

### Controllers

Use Echo's route groups and built-in binding/validation.

#### Auth endpoints (no auth required)

- `POST /api/v1/auth/register` — create user + create their first organization. Request: `{email, password, name, org_name}`. Response: JWT pair.
- `POST /api/v1/auth/login` — authenticate with email/password. Response: JWT pair.
- `POST /api/v1/auth/refresh` — exchange refresh token for new JWT pair. Request: `{refresh_token}`. Response: new JWT pair.

#### Organization endpoints (JWT auth required)

- `GET /api/v1/organizations` — list organizations the current user belongs to
- `GET /api/v1/organizations/{org_id}` — get organization details
- `PUT /api/v1/organizations/{org_id}` — update organization (admin+ only)
- `POST /api/v1/organizations` — create a new organization (current user becomes owner)

#### Member endpoints (JWT auth + tenant middleware)

- `GET /api/v1/organizations/{org_id}/members` — list members
- `POST /api/v1/organizations/{org_id}/members/invite` — invite user by email (admin+ only). If user exists, add membership. If not, create user record with null password_hash.
- `PUT /api/v1/organizations/{org_id}/members/{member_id}` — update member role (owner only)
- `DELETE /api/v1/organizations/{org_id}/members/{member_id}` — remove member (admin+ only, cannot remove last owner)

#### API Key endpoints (JWT auth + tenant middleware)

- `POST /api/v1/organizations/{org_id}/api-keys` — create API key. Returns the full key ONCE in the response. Store only the bcrypt hash.
- `GET /api/v1/organizations/{org_id}/api-keys` — list API keys (shows prefix, name, access_level, created_at, last_used_at — never the key itself)
- `DELETE /api/v1/organizations/{org_id}/api-keys/{key_id}` — revoke API key (admin+ only)

### Service Logic (`internal/services/organization/`)

Business logic layer between controllers and repositories. Controllers call service functions, service functions call repositories. Service functions contain validation, authorization checks, and orchestration logic.

### Frontend

1. **Login page** (`/auth/login`) — email + password form, calls login API, stores JWT in memory (not localStorage)
2. **Registration page** (`/auth/register`) — email, password, name, org name form
3. **Organization list** (`/`) — after login, show list of user's organizations
4. **Organization dashboard** (`/orgs/{org_id}`) — placeholder page showing org name. This will become the workspace list in Phase 3.
5. **Members page** (`/orgs/{org_id}/settings/members`) — list members, invite form, role management
6. **API Keys page** (`/orgs/{org_id}/settings/api-keys`) — list keys, create key (show secret once in modal)
7. **Auth context** — React context that holds the JWT, provides login/logout/refresh functions, auto-refreshes before expiry
8. **Protected routes** — redirect to login if not authenticated
9. Set up TanStack Query for all API calls
10. Set up TanStack Router for type-safe routing

### Tests

Write tests for:

1. **Auth flow**: register → login → access protected endpoint → refresh token → access again
2. **Tenant isolation**: create two users in different orgs. User A cannot GET/PUT/DELETE anything in User B's org. This is the most important test.
3. **RBAC enforcement**: viewer cannot invite members, member cannot change roles, admin cannot remove owner
4. **API key auth**: create key → use key to access endpoint → revoke key → access fails
5. **Edge cases**: duplicate email registration, wrong password, expired token, malformed token, missing org membership

## Constraints

- Passwords must be validated: minimum 8 characters
- JWTs must never contain the password hash or encryption key
- API key secrets must only be returned at creation time, never again
- All org-scoped queries must filter by the org_id from the JWT/tenant middleware — never trust the URL alone
- Do not add workspace tables yet (Phase 3). The `api_keys.workspace_id` column can be nullable for now.

## Verification

```bash
# Tests pass
make test

# Register a user
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"test@example.com","password":"securepass","name":"Test User","org_name":"Test Org"}'
# → returns access_token and refresh_token

# Use the token to list orgs
curl http://localhost:8080/api/v1/organizations \
  -H 'Authorization: Bearer <access_token>'
# → returns list with "Test Org"

# Frontend login page loads and works (separate dev server)
cd web && npm run dev
# → open http://localhost:5173/auth/login
```
