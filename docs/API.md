# API Reference (v1)

## Conventions

- Base path: `/v1`. JSON bodies; all IDs are UUIDs.
- Caller authentication: `Authorization: Bearer <app-api-key>` on every request (see INTEGRATION.md).
- Errors use one envelope everywhere:

```json
{ "error": { "code": "role_not_found", "message": "role does not exist in this tenant" } }
```

- List endpoints accept `limit` (default 50, max 200) and `offset`, and return `{ "data": [...], "total": n, "limit": l, "offset": o }`.
- Mutations are idempotent (FR-9): repeating a PUT assignment returns `204`, not an error.

## Authorization checks

### Check

```http
POST /v1/check
{
  "tenant_id":  "3f8a2c1e-9b4d-4e6a-8c7f-2d5b9e1a4c3d",
  "subject_id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
  "resource":   "invoice",
  "action":     "approve",
  "context":    {}
}
```

`resource` is a resource **type** (v1 is type-level; see ROADMAP.md for instance-level plans). `context` is reserved for future ABAC — send `{}` or omit.

**200:**

```json
{
  "allowed": true,
  "decision_id": "b5c7e1a2-4d3f-4b8a-9c6e-1f2a3b4c5d6e",
  "reason": "permission invoice:approve via role 'manager' (inherited from 'approver')"
}
```

A deny is also **200** with `"allowed": false` — HTTP errors are reserved for malformed/unauthenticated requests and service failures (fail-closed ⇒ `503` means "treat as DENY, may retry").

### BatchCheck

```http
POST /v1/check/batch
{ "tenant_id": "...", "subject_id": "...",
  "checks": [ { "resource": "invoice", "action": "approve" },
              { "resource": "invoice", "action": "delete" } ] }
```

**200:** `{ "results": [ { "allowed": true, ... }, { "allowed": false, ... } ] }` — order preserved.

### ListSubjectPermissions (for UI rendering)

```http
GET /v1/tenants/{tenant_id}/subjects/{subject_id}/permissions
```

**200:** `{ "roles": ["manager"], "permissions": ["invoice:approve", "invoice:read", "report:*"] }`
— the *resolved, effective* set (inheritance and wildcards included). One call renders a whole menu; never loop over Check for that.

## Management

### Roles

| Method & path | Notes |
|---|---|
| `POST /v1/tenants/{tid}/roles` | body: `{ "name", "description", "inherits": ["<role-id>", ...] }` |
| `GET /v1/tenants/{tid}/roles` · `GET .../roles/{rid}` | list / read |
| `PATCH /v1/tenants/{tid}/roles/{rid}` | rename, describe, `is_disabled`, replace `inherits` |
| `DELETE /v1/tenants/{tid}/roles/{rid}` | assignments cascade |
| `PUT /v1/tenants/{tid}/roles/{rid}/permissions/{pid}` | idempotent attach |
| `DELETE /v1/tenants/{tid}/roles/{rid}/permissions/{pid}` | detach |

`inherits` follows FR-4: the created role **receives** the listed roles' permissions (creating `manager` with `"inherits": [<viewer-id>]` ⇒ manager ⊇ viewer). Cycle-creating writes return `409 inheritance_cycle`.

### Permissions

| Method & path | Notes |
|---|---|
| `POST /v1/tenants/{tid}/permissions` | body: `{ "resource": "invoice", "action": "approve", "description" }`; `action: "*"` allowed |
| `GET /v1/tenants/{tid}/permissions` | list |
| `DELETE /v1/tenants/{tid}/permissions/{pid}` | detaches from all roles |

### Subject ↔ role assignment

| Method & path | Notes |
|---|---|
| `PUT /v1/tenants/{tid}/subjects/{sid}/roles/{rid}` | idempotent assign |
| `DELETE /v1/tenants/{tid}/subjects/{sid}/roles/{rid}` | revoke — takes effect immediately (cache version bump) |
| `GET /v1/tenants/{tid}/subjects/{sid}/roles` | list a subject's direct roles |

### Tenants (System Operator only)

`POST /v1/tenants` — body `{ "name", "admin_subject_id" }`; provisions the tenant plus its bootstrap admin role (`*:*`) per FR-10. Operator credentials required.

### Audit

```http
GET /v1/tenants/{tid}/audit-logs?from=...&to=...&subject_id=...&decision=DENY&limit=50&offset=0
```

## gRPC

```protobuf
syntax = "proto3";

package authz.v1;

option go_package = "github.com/jirugutema/rbac_service/api/authz/v1;authzv1";

service AuthorizationService {
  rpc Check(CheckRequest) returns (CheckResponse);
  rpc BatchCheck(BatchCheckRequest) returns (BatchCheckResponse);
  rpc ListSubjectPermissions(ListSubjectPermissionsRequest) returns (ListSubjectPermissionsResponse);
}

message CheckRequest {
  string tenant_id  = 1;
  string subject_id = 2;
  string resource   = 3;              // resource type
  string action     = 4;
  map<string, string> context = 5;    // reserved for ABAC (v2)
}

message CheckResponse {
  bool   allowed     = 1;
  string decision_id = 2;
  string reason      = 3;
}

message BatchCheckRequest {
  string tenant_id  = 1;
  string subject_id = 2;
  message Item { string resource = 1; string action = 2; }
  repeated Item checks = 3;
}

message BatchCheckResponse { repeated CheckResponse results = 1; }

message ListSubjectPermissionsRequest {
  string tenant_id  = 1;
  string subject_id = 2;
}

message ListSubjectPermissionsResponse {
  repeated string roles       = 1;
  repeated string permissions = 2;    // resolved "resource:action" strings
}
```

Proto and generated stubs live in the public `api/authz/v1/` path so consumer services can import the client (never under `internal/`).

## Versioning policy

- REST is versioned in the URI (`/v1`); gRPC in the proto package (`authz.v1`).
- **Additive** changes (new endpoints, new optional fields) are non-breaking and ship within v1.
- **Breaking** changes ship as `/v2` + `authz.v2` running alongside v1, with a published deprecation window before v1 is removed.
- The reserved `context` field exists precisely so ABAC can arrive without a breaking change.
