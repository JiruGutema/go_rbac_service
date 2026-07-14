# Software Requirements Specification (v1)

## 1. Purpose

The RBAC Service is a dedicated, multi-tenant authorization engine. External applications send evaluation requests and receive deterministic Allow/Deny decisions based on subjects, roles, permissions, and role inheritance.

## 2. Scope

**In scope (v1):** multi-tenant isolation, subject→role mapping, role inheritance, type-level permissions (`resource:action`), wildcard matching, decision caching, immutable audit logging, REST + gRPC APIs.

**Out of scope (v1)** — each deliberately deferred, see [ROADMAP.md](ROADMAP.md):

- Instance-level grants ("subject can edit *this specific* document")
- ABAC / policy constraints (IP ranges, time windows, attributes)
- Deny/negative grants (v1 is allow-only)
- Groups/teams as assignable units
- Authentication, identity, sessions, credentials — always external (Keycloak, Auth0, Cognito, …)

## 3. Definitions

| Term | Description |
|---|---|
| **Tenant** | An isolated logical partition (one customer/organization-unit). All data is tenant-scoped. |
| **Subject** | An entity requesting access (user, service account). Identified by an externally-issued UUID; the service stores no identity metadata. |
| **Resource** | A protected resource **type** (e.g. `invoice`, `article`). v1 does not model individual instances. |
| **Action** | An operation on a resource type (e.g. `read`, `approve`). |
| **Permission** | The pair `resource:action` (e.g. `invoice:approve`). |
| **Role** | A named, tenant-scoped collection of permissions, assignable to subjects. |

## 4. Actors

- **System Operator** — provisions tenants and operates the service. Authenticated by bootstrap/operator credentials from configuration — *not* represented as rows in tenant RBAC tables.
- **Tenant Administrator** — manages roles, permissions, and assignments inside one tenant. Created automatically when the tenant is provisioned (bootstrap flow, FR-10).
- **Client Application** — a downstream service calling the check API. Authenticated with a per-application API key (optionally mTLS).

## 5. Functional Requirements

### FR-1 Multi-tenancy
Strict logical isolation between tenants, **enforced at the database layer** (composite foreign keys — see ARCHITECTURE.md), not only in application code. No cross-tenant role, permission, assignment, or audit visibility.

### FR-2 Subject mapping
Subjects are tracked only by stable external UUIDs. No emails, names, or credentials are stored.

### FR-3 Role lifecycle
CRUD on roles within a tenant; roles can be disabled without deletion. Role names are unique per tenant.

### FR-4 Role inheritance
A role may **inherit from** one or more other roles in the same tenant. A role's *effective permissions* = its directly assigned permissions ∪ the effective permissions of every role it inherits from (transitively).

> Convention (used consistently in API, schema, and engine): `"inherits": ["<viewer-role-id>"]` on role *Manager* means **Manager ⊇ Viewer's permissions**. The inheriting role is the more powerful one.

Writes that would create a cycle are rejected. Traversal depth is capped (default 10) as a defense-in-depth limit.

### FR-5 Permissions and wildcards
Permissions are `resource:action` at the **type level**. Wildcards: `invoice:*` (all actions on a resource type) and `*:*` (tenant-wide, intended for tenant-admin roles) are valid stored permissions. Matching is engine-side. v1 is **allow-only**: any matching permission ⇒ ALLOW; no match ⇒ DENY.

### FR-6 Evaluation API
The engine evaluates `(tenant_id, subject_id, resource, action)` and returns `{allowed, decision_id, reason}`. The API provides:
- single **Check**
- **BatchCheck** (many decisions, one round trip)
- **ListSubjectPermissions** (a subject's resolved effective permissions — required by UIs to render menus without N check calls)

Requests carry a reserved optional `context` object, unused in v1, so adding ABAC later is non-breaking.

### FR-7 Caller authentication
Client applications authenticate to this service with per-application API keys; mTLS is supported for service-to-service transport. End-user JWT verification (JWKS) is an **optional, config-gated convenience mode**, disabled by default — the standard pattern is that the calling app (or its gateway) verifies the end-user JWT and forwards only the extracted `tenant_id`/`subject_id` (see INTEGRATION.md).

### FR-8 Audit logging
Every decision — **cache hit and cache miss alike** — produces an append-only audit record: timestamp, tenant, subject, resource, action, decision, reason. Audit writes are asynchronous and never block or fail a decision.

### FR-9 Idempotent mutations
Assignment operations (role→permission, subject→role, inheritance links) are idempotent: repeating an assignment is a success, not an error (`ON CONFLICT DO NOTHING` semantics).

### FR-10 Tenant provisioning & bootstrap
Tenant creation is a System Operator action. Provisioning a tenant atomically creates a default tenant-admin role (holding `*:*`) and assigns the supplied initial admin subject, solving the "who administers a fresh tenant" bootstrap problem.

## 6. Non-Functional Requirements

| ID | Requirement | Mechanism (see ARCHITECTURE.md) |
|---|---|---|
| NFR-1 | Cache-hit decisions < 3 ms; cache-miss < 10 ms p99 | Redis-resolved permission sets; single recursive query on miss |
| NFR-2 | ≥ 15,000 checks/sec per node baseline | Stateless nodes, horizontal scale-out |
| NFR-3 | 99.99% availability target | HA Postgres/Redis, stateless engine |
| NFR-4 | **Fail closed** | Any unexpected internal failure ⇒ DENY, via the degradation ladder |
| NFR-5 | Stateless compute nodes | All state in Postgres/Redis; HPA-friendly |
| NFR-6 | TLS 1.3 on all edges; mTLS for internal calls; AES-256 at rest | Deployment requirement |
| NFR-7 | Error responses never leak cross-tenant existence information | Uniform 404/403 behavior |
