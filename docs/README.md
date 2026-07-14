# Centralized Authorization Service: Project Blueprint

This blueprint combines a **Software Requirements Specification (SRS)** and a **Software Architecture Document (SAD)** for an enterprise-ready, standalone Role-Based Access Control (RBAC) service. It is designed to scale from an embedded module within a monolith into a highly distributed, decoupled authorization microservice inspired by modern systems like OpenFGA, Auth0, and Ory Keto.

---

# Document 1: Software Requirements Specification (SRS)

## 1. Introduction

### 1.1 Purpose

The purpose of this document is to specify the functional, non-functional, and interface requirements for the centralized **RBAC (Role-Based Access Control) Service**. The RBAC Service acts as a dedicated, high-performance, and multi-tenant authorization engine. It provides external microservices and applications with deterministic authorization decisions (Allow/Deny) based on subjects, roles, permissions, and dynamic policies.

### 1.2 Scope

The system acts exclusively as an **Authorization-as-a-Service** platform.

* **In Scope:** Centralized identity mapping (by Subject ID), role hierarchy traversal, multi-tenant workspace isolation, fine-grained permission assignments, cryptographic policy token parsing, policy evaluation cache layers, and immutable audit trailing.
* **Out of Scope:** User authentication, identity verification, session management, and credential/password storage. Authenticators like Keycloak, Auth0, Okta, or AWS Cognito must handle authentication and supply identity contexts via secure JSON Web Tokens (JWTs).

### 1.3 Definitions and Acronyms

| Term | Acronym / Alias | Description |
| --- | --- | --- |
| **Subject** | Principal | An entity (user, service account, or background process) requesting access to a resource. |
| **Resource** | Object | A protected digital asset or entity within the system domain (e.g., `file`, `tenant-setting`, `project`). |
| **Action** | Operation | The execution vector being attempted on a resource (e.g., `read`, `write`, `delete`, `approve`). |
| **Permission** | Privilege | The explicit coupling of an Action and a Resource (Format: `resource:action`). |
| **Role** | Group Profile | A named collection of permissions that can be assigned to one or more subjects. |
| **Policy** | Evaluation Rule | Contextual or conditional constraints (such as IP limits, time windows, or attributes) applied to authorization. |
| **Tenant** | Organization | A completely isolated logical partition within the system representing a distinct client or enterprise. |

---

## 2. Overall Description

### 2.1 User and System Actors

* **Global Administrator:** System-wide operators responsible for multi-tenant provisioning, baseline schema definitions, global policy creation, and system monitoring.
* **Organization Administrator:** Tenant-specific users empowered to manage localized roles, map users to local roles, and inspect tenant-scoped audit logs.
* **Client Application / Microservice:** Downstream consumers that programmatically dispatch access evaluation requests to the RBAC service to guard their internal endpoints and data layers.
* **Authorization Middleware:** Low-latency interceptors placed at the API Gateway or reverse-proxy level that query the RBAC engine to authenticate request paths before routing.

### 2.2 System Use Cases

#### Use Case 1: Administrative Policy and Role Mapping

* **Primary Actor:** Organization Administrator
* **Flow:** The administrator requests the creation of a custom role (`Editor`), assigns a predefined block of permissions (`article:write`, `article:publish`), and associates this role with a list of Subject UUIDs.
* **Postconditions:** Data is immediately persisted to the write-optimized database, and affected cached entries are invalidated via a publish-subscribe event.

#### Use Case 2: Programmatic Access Evaluation

* **Primary Actor:** Authorization Middleware / Microservice
* **Flow:** A service receives a payload from a user and dispatches a synchronous evaluation payload (`POST /v1/authorize`) containing the Subject ID, requested Action, and target Resource identifier.
* **Postconditions:** The engine resolves the subject's roles, processes inheritance chains, evaluates structural policies, writes an entry to the immutable audit log, and returns a boolean result along with execution metadata.

---

## 3. Functional Requirements

### 3.1 Tenant & User Space Isolation

* **FR-1.1 (Multi-Tenancy):** The system **shall** enforce strict logical isolation between tenants. No subject, role, permission, or audit log from Tenant $A$ shall be visible or evaluable within the scope of Tenant $B$.
* **FR-1.2 (Subject Mapping):** The system **shall** track subjects using stable, immutable identifiers (such as UUIDv4 or ULID). It will not manage identity metadata like emails or passwords.

### 3.2 Role and Hierarchical Model Management

* **FR-2.1 (Role Lifecycle):** The system **shall** support standard CRUD operations on roles, including capabilities to clone, provision, disable, or enable roles within a tenant space.
* **FR-2.2 (Hierarchical Inheritance):** The system **shall** support directional role inheritance graph models (e.g., `SuperAdmin` $\rightarrow$ `Admin` $\rightarrow$ `Editor` $\rightarrow$ `Viewer`). Higher-tier roles automatically inherit all explicit permissions assigned to their descendants.

### 3.3 Fine-Grained Permission Schema

* **FR-3.1 (Structure):** Permissions **must** be expressed deterministically in the form `resource:action` (e.g., `document:share`).
* **FR-3.2 (Wildcard Matrix):** The engine **shall** support wildcard syntax matching for comprehensive resource operations (e.g., `billing:*` implies access to all actions under the billing domain).

### 3.4 Authorization Evaluation & Engine Context

* **FR-4.1 (Evaluation Vector):** The authorization interface **must** process evaluation tuples containing `(TenantID, SubjectID, Action, Resource)`.
* **FR-4.2 (Policy Constraints):** The engine **shall** support dynamic rule validation including Attribute-Based Access Control (ABAC) extensions, checking parameters such as client IP range boundaries, temporal activation frames, or environmental variables.

### 3.5 Token Interception and Integration

* **FR-5.1 (JWT Verification):** The authorization service **shall** be capable of verifying incoming JSON Web Tokens using configured asymmetric key sets (JWKS) to extract inline claims, such as implicit tenant IDs or user contexts.

### 3.6 Diagnostic Audit Logging

* **FR-6.1 (Immutability):** Every authorization query **must** generate an unalterable, structured trace tracking the request signature: timestamp, tenant, subject, resource, action, decision outcome (`ALLOW`/`DENY`), and evaluation reasoning metadata.

---

## 4. Non-Functional Requirements

### 4.1 Performance & Low-Latency Target Metrics

* **NFR-1.1 (Latencies):** Cache-hit authorization decisions **shall** resolve in under **3ms**. Cache-miss evaluations requiring direct relational queries must resolve in under **10ms** at the 99th percentile ($p99$).
* **NFR-1.2 (Throughput):** The engine **shall** handle a minimum baseline of 15,000 requests per second (RPS) per cluster node via scale-out deployment designs.

### 4.2 Availability & Resilience Architecture

* **NFR-2.1 (Uptime SLA):** The production deployment architecture **must** maintain a minimum highly available threshold of **99.99%** uptime annually.
* **NFR-2.2 (Fault Isolation):** Database or caching cluster dropouts **shall** trigger safe failover mechanisms. The engine will gracefully degrade to deterministic local configurations or reject access with a safe error state rather than exposing unprotected access vectors.

### 4.3 Scalability

* **NFR-3.1 (Stateless Computation):** The core engine nodes **must** remain entirely stateless, allowing automatic horizontal scaling via Kubernetes Horizontal Pod Autoscalers (HPA) driven by CPU and network saturation metrics.

### 4.4 Comprehensive Security Measures

* **NFR-4.1 (Transport Encryption):** All edge connections, internode communications, and database configurations **must** require TLS 1.3 encryption. Internal microservice integrations must leverage mutual TLS (mTLS).
* **NFR-4.2 (Data-at-Rest Protection):** Storage buckets, backup objects, and active relational schemas **must** be encrypted at rest using AES-256 standards.

---

## 5. Interface & API Specifications

### 5.1 RESTful Endpoint Interfaces

#### Create a New Role

```http
POST /v1/tenants/{tenant_id}/roles
Content-Type: application/json

{
  "name": "Manager",
  "description": "Mid-level regional operations manager",
  "parent_roles": ["Viewer"]
}

```

#### Execute Runtime Authorization Check

```http
POST /v1/authorize
Content-Type: application/json

{
  "tenant_id": "org_710a30b4_ef91",
  "subject": "usr_01h7xzz94",
  "resource": "invoice:inv_99823",
  "action": "approve"
}

```

**Response ($200\text{ OK}$):**

```json
{
  "allowed": true,
  "decision_id": "log_01h7y00a8b",
  "reason": "Explicit permission assigned via inherited role 'Manager'"
}

```

### 5.2 gRPC Protocol Buffer Definition

```protobuf
syntax = "proto3";

package authz.v1;

option go_package = "internal/adapters/grpc/proto;v1";

service AuthorizationService {
  rpc Authorize(AuthorizeRequest) returns (AuthorizeResponse);
}

message AuthorizeRequest {
  string tenant_id = 1;
  string subject   = 2;
  string resource  = 3;
  string action    = 4;
}

message AuthorizeResponse {
  bool allowed      = 1;
  string decision_id = 2;
  string reason     = 3;
}

```

---

# Document 2: Software Architecture Document (SAD)

## 1. Architectural Styles and Strategy

The RBAC Service is structured using **Hexagonal Architecture (Ports and Adapters)** integrated with **Clean Architecture** principles.

```
                  ┌─────────────────────────────────────────┐
                  │               Presentation              │
                  │         (HTTP REST / gRPC Core)         │
                  └────────────────────┬────────────────────┘
                                       │
                                       ▼
                  ┌─────────────────────────────────────────┐
                  │            Application Layer            │
                  │         (Use Cases & Orchestration)     │
                  └────────────────────┬────────────────────┘
                                       │
                                       ▼
                  ┌─────────────────────────────────────────┐
                  │               Domain Core               │
                  │        (Entities & Policy Engine)       │
                  └────────────────────┬────────────────────┘
                                       │
                                       ▼
                  ┌─────────────────────────────────────────┐
                  │          Infrastructure Adapters        │
                  │       (PostgreSQL / Redis / Slog)       │
                  └─────────────────────────────────────────┘

```

### Rationale

* **Independence of Infrastructure:** The central policy execution code contains pure authorization logic. It remains completely unaware of whether storage is managed via PostgreSQL, a Spanner cluster, or flat files.
* **Deterministic Isolation for Testing:** Business flows, role hierarchies, and policy evaluation rules can be comprehensively tested with mock adapters without initializing databases or network listeners.

---

## 2. Relational Database Schema Design

The backing datastore uses a normalization scheme tailored for multi-tenant isolation, quick relational indexing, and referential integrity constraints.

```
 ┌──────────────────────┐        ┌──────────────────────┐        ┌──────────────────────┐
 │     organizations    │        │         users        │        │         roles        │
 ├──────────────────────┤        ├──────────────────────┤        ├──────────────────────┤
 │ PK  id (UUID)        │◄───────┤ PK  id (UUID)        │◄───┐   │ PK  id (UUID)        │◄───┐
 │     name             │        │ FK  org_id           │    │   │ FK  org_id           │    │
 └──────────────────────┘        └──────────────────────┘    │   │     name             │    │
                                                             │   └──────────────────────┘    │
                                 ┌──────────────────────┐    │                               │
                                 │      user_roles      │    │   ┌──────────────────────┐    │
                                 ├──────────────────────┤    │   │   role_permissions   │    │
                                 │ PK/FK user_id        ├────┘   ├──────────────────────┤    │
                                 │ PK/FK role_id        ├───────►│ PK/FK role_id        ├────┘
                                 └──────────────────────┘        │ PK/FK permission_id  ├───┐
                                                                 └──────────────────────┘   │
                                 ┌──────────────────────┐                                   │
                                 │     permissions      │                                   │
                                 ├──────────────────────┤                                   │
                                 │ PK  id (UUID)        │◄──────────────────────────────────┘
                                 │     resource         │
                                 │     action           │
                                 └──────────────────────┘

```

### Relational Schema Definition (DDL)

```sql
CREATE TABLE organizations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TABLE roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_org_role_name UNIQUE(org_id, name)
);

CREATE TABLE role_hierarchy (
    parent_role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    child_role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    PRIMARY KEY (parent_role_id, child_role_id),
    CONSTRAINT no_self_inheritance CHECK (parent_role_id <> child_role_id)
);

CREATE TABLE permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    resource VARCHAR(100) NOT NULL,
    action VARCHAR(100) NOT NULL,
    CONSTRAINT unique_org_permission UNIQUE(org_id, resource, action)
);

CREATE TABLE role_permissions (
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

CREATE TABLE user_roles (
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    PRIMARY KEY (org_id, user_id, role_id)
);

```

---

## 3. Runtime Access Verification Flow

```
[Client App] ──(1) Evaluates Route──► [API Gateway / Middleware]
                                             │
                                      (2) POST /authorize
                                             │
                                             ▼
                                     [Core Engine Node]
                                             │
                                    (3) Query Redis Cache
                                             │
                       ┌─────────────────────┴─────────────────────┐
                       ▼ (Cache Hit)                               ▼ (Cache Miss)
               [Return Decision]                           [Fetch Dependency Graph]
                                                                   │
                                                           • Load User Roles
                                                           • Recurse Role Hierarchies
                                                           • Extract Resource Rules
                                                                   │
                                                                   ▼
                                                           [Evaluate Logic Engine]
                                                                   │
                                                           • Write to Redis Cache
                                                           • Stream to Async Audit
                                                                   │
                                                                   ▼
                                                           [Return Decision]

```

---

## 4. Codebase Directory Layout

The workspace structure adheres to canonical Go layout standards for enterprise services, strictly decoupling configuration and wiring tasks from actual business domains.

```text
.
├── cmd
│   └── authz-server                     # Application entry point (main.go initialization)
├── configs                              # Default system configurations and environment templates
├── deployments                          # Infrastructure descriptors (Dockerfiles, Helm charts)
├── migrations                           # Explicit, sequential SQL migration schemas
└── internal
    ├── domain                           # Core Enterprise Business Domain Rules (Strictly Pure Go)
    │   ├── models.go                    # Immutable structural definitions (Role, Permission, Tenant)
    │   └── engine.go                    # Core policy evaluation execution logic
    ├── application                      # Business Use Case Interactors and Transaction Boundaries
    │   ├── authorize.go                 # Access decision command processor
    │   ├── manage_roles.go              # Business logic validation for role management
    │   └── ports.go                     # Driving and Driven Abstract Interface Signatures
    └── adapters                         # Concrete Low-Level External Bindings (Infrastructure)
        ├── db                           # PostgreSQL implementations and repository maps
        ├── cache                        # Redis lookup operations and invalidation logic
        ├── http                         # REST server routers, payload models, and middleware
        └── grpc                         # Core gRPC server listeners and handlers

```

---

## 5. Technology Blueprint Matrix

| Component Layer | Selection Standard | Architecture Justification |
| --- | --- | --- |
| **Language Runtime** | Go 1.24+ | Exceptional operational performance, minimal memory footprints, native concurrency, and robust handling of high throughput network workloads. |
| **API Foundations** | Standard `net/http` + gRPC | Minimizes external dependencies for REST configurations, using gRPC to achieve low-overhead internal mesh communication. |
| **Primary Datastore** | PostgreSQL | Strong relational query engines, robust handling of concurrent reads, and strict ACID compliance for managing critical access matrices. |
| **Relational Binder** | `sqlc` | Provides compile-time type safety by generating clean, performance-optimized, type-safe Go code directly from raw SQL schemas. |
| **Distributed Cache** | Redis | High-performance in-memory key-value lookups, allowing authorization vectors to bypass disk reads and meet low-latency targets. |
| **Telemetry Pipeline** | OpenTelemetry | Vendor-neutral tracing instrumentation, enabling step-by-step performance analysis across complex distributed graphs. |
| **Structured Logging** | Standard `log/slog` | High-performance JSON logging included in the standard library, avoiding heavy external logger frameworks. |

---

## 6. Implementation Best Practices

1. **Deterministic Cache Invalidation:** Write operations targeting roles, users, or permissions must use a transactional database hook or outbox pattern to broadcast invalidation signals to all active Redis cache rings.
2. **Fail-Closed Security Design:** If any internal component encounters unexpected edge failures (such as runtime engine context cancellations, driver dropouts, or buffer overflows), the evaluation pipeline must return a definitive and secure access decision: **`DENY`**.
3. **Idempotence Across Commands:** Mutation APIs (such as assigning permissions or linking roles) must support idempotent operations, ensuring that processing duplicate messages does not trigger data layer errors. Use safe conditional upserts (`ON CONFLICT DO NOTHING`) to safely manage race conditions.
