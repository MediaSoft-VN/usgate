# White-label Multi-tenant M1 — Design

**Version:** 1.0
**Scope:** Tenant, custom domain, membership and tenant RBAC
**Status:** Design approved for implementation

## 1. Objective

Allow any eligible user to operate an isolated White-label tenant with its own
domain, members and permissions. M1 establishes the security boundary. Wallet,
payment channels and tenant branding are later milestones that consume this
boundary.

## 2. Scope and non-goals

### In scope

- Tenant lifecycle and status.
- User memberships and tenant roles.
- Verified custom domains.
- Hostname-to-tenant resolution.
- Tenant-aware authorization and audit events.
- Cross-tenant isolation tests.

### Out of scope for M1

- Tenant wallets and payment channels.
- Provider/channel ownership migration.
- Custom branding configuration.
- Tenant-specific API credentials.
- Billing settlement.

## 3. Domain model

### `tenants`

| Field | Rule |
|---|---|
| `id` | Primary key |
| `slug` | Lowercase, normalized, unique, max 64 |
| `name` | Required, max 128 |
| `status` | `active`, `suspended`, `deleted` |
| `created_by` | Creating user ID |
| `created_at`, `updated_at` | Unix seconds |
| `deleted_at` | Nullable soft-delete timestamp |

### `tenant_memberships`

| Field | Rule |
|---|---|
| `tenant_id`, `user_id` | Indexed, unique together |
| `role_key` | `owner`, `admin`, `member`, `viewer` |
| `status` | `active`, `invited`, `suspended`, `removed` |
| timestamps | Created, updated, last seen |

There must be exactly one active owner during normal operation. The final
active owner cannot be removed or demoted.

### `tenant_domains`

| Field | Rule |
|---|---|
| `tenant_id` | Indexed |
| `hostname` | Lowercase ASCII hostname, globally unique |
| `status` | `pending`, `verified`, `disabled` |
| `verification_method` | `dns_txt` in M1 |
| `verification_token_hash` | Hash only; never store raw token |
| `verified_at`, `last_checked_at` | Nullable timestamps |

## 4. Request resolution

1. Authenticate the request when the route requires authentication.
2. Normalize the trusted hostname: lowercase, remove trailing dot, reject
   malformed labels and user-info syntax.
3. Resolve only an exact verified domain.
4. Load tenant status and the current user's membership.
5. Put a typed tenant context into the request.
6. Enforce the tenant permission before the controller executes.

Untrusted `X-Forwarded-Host` must never select a tenant. A client-supplied
tenant ID cannot override a verified hostname. Missing context fails closed.

## 5. Roles and permissions

| Role | Capabilities |
|---|---|
| `owner` | Full tenant administration and ownership transfer |
| `admin` | Member and domain administration |
| `member` | Normal tenant operations |
| `viewer` | Read-only tenant access |

Suggested permissions:

```text
tenant.read
tenant.update
tenant.delete
tenant.members.read
tenant.members.invite
tenant.members.update
tenant.members.remove
tenant.domains.read
tenant.domains.create
tenant.domains.verify
tenant.domains.disable
```

The existing global platform role remains separate from tenant membership.

## 6. API design

### Tenant lifecycle

```text
GET   /api/tenants
GET   /api/tenants/:tenant_id
POST  /api/tenants
PATCH /api/tenants/:tenant_id
```

Create request:

```json
{"slug":"acme","name":"Acme"}
```

Creation atomically creates the tenant and the owner membership.

### Memberships

```text
GET    /api/tenants/:tenant_id/members
POST   /api/tenants/:tenant_id/members/invitations
PATCH  /api/tenants/:tenant_id/members/:user_id
DELETE /api/tenants/:tenant_id/members/:user_id
POST   /api/tenants/:tenant_id/members/:user_id/transfer-owner
```

Ownership transfer requires recent authentication/step-up verification.

### Domains

```text
GET    /api/tenants/:tenant_id/domains
POST   /api/tenants/:tenant_id/domains
POST   /api/tenants/:tenant_id/domains/:domain_id/verify
DELETE /api/tenants/:tenant_id/domains/:domain_id
```

Domain creation returns a DNS TXT challenge. Raw challenge tokens are shown
only at creation and are excluded from logs and audit records.

### Context

```text
GET  /api/tenant-context
POST /api/tenant-context/select
```

Explicit tenant selection is allowed only for an authenticated user with an
active membership and cannot conflict with a verified request hostname.

## 7. Security rules

- Enforce tenant isolation on the server, never in the frontend.
- Query tenant resources with the resolved `tenant_id` in every operation.
- Do not disclose foreign tenant existence through error differences.
- Hash invitation and domain verification tokens.
- Rate-limit domain verification and invitations.
- Audit actor, tenant, domain, action and result; never log usable secrets.
- Require step-up verification for ownership transfer and tenant deletion.
- Invalidate cached membership/context after role or membership changes.

## 8. Migration

1. Add the three tables and indexes using GORM-compatible migrations.
2. Seed a deterministic `default` tenant.
3. Add existing root users as owners, admins as admins and other enabled users
   as members.
4. Run migration twice to prove idempotency.
5. Keep existing global routes unchanged until later resource migrations.
6. Verify fresh and representative upgraded databases on SQLite, MySQL and
   PostgreSQL.

## 9. Test plan

- Tenant creation atomically creates owner membership.
- Duplicate slug/domain is rejected.
- Owner invariants prevent removing the final owner.
- Verified hostname resolves to the correct tenant.
- Pending, disabled, expired and mismatched domains are denied.
- Tenant A cannot read, update or delete Tenant B resources.
- A user's role in Tenant A does not affect Tenant B.
- Membership removal takes effect with an existing session.
- Host case/trailing-dot normalization is consistent.
- Untrusted forwarded host cannot select a tenant.
- Unauthorized, non-member and suspended-tenant requests fail closed.
- Migration is idempotent on all supported database dialects.

## 10. Implementation traceability

| Design area | Planned code boundary |
|---|---|
| Models/migration | `model/tenant*.go`, `model/main.go` |
| Context | `constant/context_key.go`, `middleware/tenant.go`, `service/tenant/` |
| Authorization | `service/authz/` |
| APIs | `controller/tenant.go`, `router/api-router.go` |
| Audit | Existing audit middleware/model |
| Tests | model, service, middleware, controller and E2E test suites |
