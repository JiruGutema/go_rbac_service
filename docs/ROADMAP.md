# Roadmap

v1 is deliberately small: multi-tenant, type-level RBAC done correctly. Everything below is sequenced so none of it breaks the v1 API — which is why `CheckRequest` already reserves `context`, and why the versioning policy in [API.md](API.md) exists.

## v2 — Instance-level authorization (ReBAC)

The headline v2 feature: answering "can subject X approve **invoice 99823**", not just "invoices in general".

- **Model:** Zanzibar-style relationship tuples — `(tenant, subject, relation, resource_type, resource_id)` — the approach proven by OpenFGA, SpiceDB, and Ory Keto. A new `relationships` table; the engine unions type-level RBAC (v1 semantics, unchanged) with instance-level tuples.
- **API:** `CheckRequest` gains an optional `resource_id`; absent ⇒ exact v1 behavior. New tuple write/read endpoints.
- **Consequences to design for:** per-resource cache keys (the v1 per-subject set no longer suffices alone) and a `ListAccessibleResources` lookup API (see below).

## v2 — ABAC / conditional policies

- **Condition language:** CEL — the choice of OpenFGA, SpiceDB, and Google; small, deterministic, non-Turing-complete.
- **Storage:** a `policies` table attaching conditions to role or permission grants (e.g. IP range, time window).
- **API:** activates the reserved `context` field (`map<string,string>`) on Check — clients pass request attributes; the engine evaluates conditions against them. Explicit evaluation semantics to be specified before implementation (proposal: conditions are grant-scoped filters; still allow-only).

## Later

- **Groups/teams** — assign roles to groups, subjects to groups (one indirection level; Zanzibar's usersets generalize this).
- **`ListAccessibleResources`** — "which invoices can this subject see?" for list-endpoint filtering; the hardest API to scale (OpenFGA `ListObjects` / SpiceDB `LookupResources` / Cerbos `PlanResources` are the prior art).
- **`Expand` / explain API** — full decision trees for debugging, beyond the `reason` string.
- **Watch / changes feed** — stream RBAC mutations (backed by the existing `outbox` table) for consumers that maintain local projections.
- **Embedded Go library mode** — import the application layer directly (no network hop) for Go monoliths; the hexagonal split exists to make this possible.
- **JWKS verification mode** — the config-gated convenience mode from FR-7, for gateway-less deployments.
- **Official client SDKs** — thin Python (PyPI), .NET (NuGet), and Go packages wrapping `api/authz/v1`.

## Non-goals (permanent)

- Authentication, identity storage, sessions, credentials — always someone else's job (Keycloak, Auth0, Cognito).
- A general policy language for arbitrary code execution — conditions stay declarative and bounded (CEL only).
