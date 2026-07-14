# Software Architecture Document (v1)

## 1. Architectural style

Hexagonal (Ports & Adapters) with Clean Architecture layering. The policy engine is pure Go with no knowledge of Postgres, Redis, or transport.

```
        ┌─────────────────────────────────────────┐
        │               Presentation              │
        │          (HTTP REST / gRPC)             │
        └────────────────────┬────────────────────┘
                             ▼
        ┌─────────────────────────────────────────┐
        │            Application Layer            │
        │        (Use cases & orchestration)      │
        └────────────────────┬────────────────────┘
                             ▼
        ┌─────────────────────────────────────────┐
        │               Domain Core               │
        │        (Entities & policy engine)       │
        └────────────────────┬────────────────────┘
                             ▼
        ┌─────────────────────────────────────────┐
        │          Infrastructure Adapters        │
        │        (PostgreSQL / Redis / slog)      │
        └─────────────────────────────────────────┘
```

**Why:** the engine can be tested with in-memory fakes (no DB, no listeners), and storage/transport can be swapped without touching authorization logic.

## 2. Database schema

Design rules baked into the DDL:

1. **Tenant isolation is enforced by the database.** Every tenant-scoped table carries `tenant_id`, and every junction table uses **composite foreign keys** `(x_id, tenant_id) → parent(id, tenant_id)`. A buggy INSERT physically cannot link rows across tenants.
2. **Subjects have no table.** Identity is external; `subject_roles.subject_id` is a bare UUID by design.
3. **Audit must survive everything.** `audit_logs` has no FK to `tenants` and is range-partitioned by time; tenants are soft-deleted (`deleted_at`), never cascaded away.
4. **Inheritance direction:** `role_inheritance.role_id` **inherits from** `inherits_from_role_id` (Manager row → Viewer row means Manager ⊇ Viewer). Cycles are rejected in the application layer on write; the recursive query also carries a depth cap as defense.

```sql
CREATE TABLE tenants (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name          VARCHAR(255) NOT NULL,
    rbac_version  BIGINT NOT NULL DEFAULT 1,   -- bumped on every RBAC mutation; drives cache keys
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at    TIMESTAMPTZ                  -- soft delete
);

CREATE TABLE roles (
    id          UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    name        VARCHAR(100) NOT NULL,
    description TEXT,
    is_disabled BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id),
    UNIQUE (id, tenant_id),                    -- composite-FK target
    UNIQUE (tenant_id, name)
);

CREATE TABLE permissions (
    id          UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    resource    VARCHAR(100) NOT NULL,         -- resource TYPE, e.g. 'invoice'
    action      VARCHAR(100) NOT NULL,         -- e.g. 'approve', or '*'
    description TEXT,
    PRIMARY KEY (id),
    UNIQUE (id, tenant_id),
    UNIQUE (tenant_id, resource, action)
);

CREATE TABLE role_inheritance (
    tenant_id             UUID NOT NULL,
    role_id               UUID NOT NULL,       -- the inheriting (more powerful) role
    inherits_from_role_id UUID NOT NULL,
    PRIMARY KEY (role_id, inherits_from_role_id),
    FOREIGN KEY (role_id, tenant_id)
        REFERENCES roles (id, tenant_id) ON DELETE CASCADE,
    FOREIGN KEY (inherits_from_role_id, tenant_id)
        REFERENCES roles (id, tenant_id) ON DELETE CASCADE,
    CHECK (role_id <> inherits_from_role_id)
);

CREATE TABLE role_permissions (
    tenant_id     UUID NOT NULL,
    role_id       UUID NOT NULL,
    permission_id UUID NOT NULL,
    PRIMARY KEY (role_id, permission_id),
    FOREIGN KEY (role_id, tenant_id)
        REFERENCES roles (id, tenant_id) ON DELETE CASCADE,
    FOREIGN KEY (permission_id, tenant_id)
        REFERENCES permissions (id, tenant_id) ON DELETE CASCADE
);

CREATE TABLE subject_roles (
    tenant_id  UUID NOT NULL,
    subject_id UUID NOT NULL,                  -- external identity; intentionally no FK
    role_id    UUID NOT NULL,
    PRIMARY KEY (tenant_id, subject_id, role_id),
    FOREIGN KEY (role_id, tenant_id)
        REFERENCES roles (id, tenant_id) ON DELETE CASCADE
);

CREATE TABLE audit_logs (
    id         UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id  UUID NOT NULL,                  -- no FK: audit outlives tenant deletion
    subject_id UUID NOT NULL,
    resource   VARCHAR(100) NOT NULL,
    action     VARCHAR(100) NOT NULL,
    decision   VARCHAR(5) NOT NULL CHECK (decision IN ('ALLOW', 'DENY')),
    reason     TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

CREATE TABLE outbox (                          -- transactional outbox for follow-up work
    id           BIGSERIAL PRIMARY KEY,
    event_type   VARCHAR(50) NOT NULL,
    payload      JSONB NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMPTZ
);
```

Hot-path indexes: `subject_roles` PK already serves "roles of (tenant, subject)"; add `role_permissions(role_id)` and `role_inheritance(role_id)` (covered by their PKs) — the whole cache-miss path is one recursive CTE plus one join.

## 3. Evaluation strategy

Cache miss resolves the subject's **effective permission set**:

1. Load role IDs from `subject_roles` for `(tenant_id, subject_id)`.
2. Expand inherited roles with a recursive CTE over `role_inheritance` (depth cap 10, disabled roles excluded).
3. Join `role_permissions` → `permissions` to produce the set of `resource:action` strings (wildcards stored as-is).
4. The engine matches the requested `resource:action` against the set, honoring `resource:*` and `*:*`.

If p99 targets are ever missed, the documented escalation is a **closure table** (materialized transitive inheritance) — not ad-hoc caching in app code.

## 4. Caching & consistency

**What is cached:** the resolved per-subject permission set (small, bounded), *not* individual decisions (unbounded cardinality, impossible to invalidate correctly).

**Key:** `authz:{tenant_id}:{rbac_version}:{subject_id}` → set of permission strings, TTL 30–60 s.

**Invalidation is deterministic, not pub/sub:** every RBAC mutation increments `tenants.rbac_version` **in the same transaction**. New reads compute new keys; stale keys expire via TTL. A missed message cannot cause a stale ALLOW because there are no messages to miss. (This is the type-level analogue of Permify's snap tokens / SpiceDB's quantized revisions; Redis pub/sub alone is fire-and-forget and unsafe for revocation.)

The `outbox` table exists for follow-up effects that must not be lost (e.g. future watch/changes feed, webhooks) — processed by a background worker, at-least-once.

## 5. Runtime flow (audit on both branches)

```
[Client App] ──(1) POST /v1/check──► [Engine Node]
                                          │
                                 (2) Redis GET authz:{t}:{v}:{s}
                                          │
                     ┌────────────────────┴────────────────────┐
                     ▼ hit                                     ▼ miss
             [Match in cached set]                  [Recursive SQL resolve]
                     │                                         │
                     │                              [Match + Redis SET (TTL)]
                     │                                         │
                     ├──────────► (3) async audit ◄────────────┤
                     ▼                                         ▼
              [Return decision]                         [Return decision]
```

Audit records are pushed to an in-process buffered writer (batch-inserted into `audit_logs`); a decision is never delayed or failed by auditing.

## 6. Failure behavior — fail-closed degradation ladder

| Failure | Behavior |
|---|---|
| Redis down | Fall through to Postgres behind a circuit breaker — latency SLO degrades; correctness doesn't. No DENY-storm. |
| Postgres down (and cache miss) | **DENY** + `503`; clients may retry. |
| Engine panic / context cancellation / anything unexpected | **DENY**. |
| Audit pipeline backlog | Decisions continue; backlog metric alarms. Audit is durable-eventually, never blocking. |

## 7. Codebase layout (target)

```text
.
├── api
│   └── authz
│       └── v1                    # .proto + generated stubs — PUBLIC importable path
├── cmd
│   └── authz-server              # package main
├── config                        # env config, DB/Redis constructors
├── migrations                    # sequential SQL migrations
└── internal
    ├── domain                    # entities + pure policy engine
    ├── application               # use cases, ports (interfaces)
    └── adapters                  # postgres, redis, http, grpc
```

> Generated gRPC code must **not** live under `internal/` — Go forbids importing it from other modules, and consumer services need the client stubs. `go_package` must be the full module path: `github.com/jirugutema/rbac_service/api/authz/v1;authzv1`.

> The current scaffold is a flattened starter (`main.go`, `config/`, `server/`, `domain/`, `handler/`, `repository/`, `service/`); migrate toward the target layout as the implementation grows.

## 8. Technology matrix

| Layer | Choice | Why |
|---|---|---|
| Runtime | Go 1.25+ | Throughput, small footprint, native concurrency |
| APIs | `net/http` + gRPC | Few dependencies; gRPC for low-overhead service-to-service |
| Datastore | PostgreSQL | ACID, composite FKs for tenant isolation, recursive CTEs |
| SQL binding | `sqlc` | Compile-time-checked queries from raw SQL |
| Cache | Redis | Sub-ms permission-set lookups |
| Telemetry | OpenTelemetry | Vendor-neutral tracing |
| Logging | `log/slog` | Structured JSON, stdlib |

## 9. Prior art

- **Slack's internal role service** — a central gRPC RBAC service consulted on every action, with non-authoritative edge caching: the closest published analogue of this design. ([slack.engineering](https://slack.engineering/role-management-at-slack/))
- **Permify** — tenant-first API design and Postgres-native snapshot tokens informed the cache-versioning scheme. ([docs.permify.co](https://docs.permify.co/operations/snap-tokens))
- **OpenFGA / SpiceDB / Zanzibar** — the relationship-tuple model v2 will adopt for instance-level authorization. ([openfga.dev](https://openfga.dev), [authzed.com](https://authzed.com/docs), [Zanzibar paper](https://www.usenix.org/system/files/atc19-pang.pdf))
