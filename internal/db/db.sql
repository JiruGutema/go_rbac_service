CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    rbac_version BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    is_disabled BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_roles_tenant
        FOREIGN KEY (tenant_id)
        REFERENCES tenants(id)
        ON DELETE CASCADE,

    CONSTRAINT uq_role_name
        UNIQUE (tenant_id, name)
);

CREATE TABLE permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    resource TEXT NOT NULL,
    action TEXT NOT NULL,
    description TEXT,

    CONSTRAINT fk_permissions_tenant
        FOREIGN KEY (tenant_id)
        REFERENCES tenants(id)
        ON DELETE CASCADE,

    CONSTRAINT uq_permission
        UNIQUE (tenant_id, resource, action)
);

CREATE TABLE role_inheritance (
    tenant_id UUID NOT NULL,
    role_id UUID NOT NULL,
    inherits_from_role_id UUID NOT NULL,

    PRIMARY KEY (tenant_id, role_id, inherits_from_role_id),

    FOREIGN KEY (tenant_id)
        REFERENCES tenants(id)
        ON DELETE CASCADE,

    FOREIGN KEY (role_id)
        REFERENCES roles(id)
        ON DELETE CASCADE,

    FOREIGN KEY (inherits_from_role_id)
        REFERENCES roles(id)
        ON DELETE CASCADE,

    CHECK (role_id <> inherits_from_role_id)
);

CREATE TABLE role_permissions (
    tenant_id UUID NOT NULL,
    role_id UUID NOT NULL,
    permission_id UUID NOT NULL,

    PRIMARY KEY (tenant_id, role_id, permission_id),

    FOREIGN KEY (tenant_id)
        REFERENCES tenants(id)
        ON DELETE CASCADE,

    FOREIGN KEY (role_id)
        REFERENCES roles(id)
        ON DELETE CASCADE,

    FOREIGN KEY (permission_id)
        REFERENCES permissions(id)
        ON DELETE CASCADE
);

CREATE TABLE subject_roles (
    tenant_id UUID NOT NULL,
    subject_id UUID NOT NULL,
    role_id UUID NOT NULL,

    PRIMARY KEY (tenant_id, subject_id, role_id),

    FOREIGN KEY (tenant_id)
        REFERENCES tenants(id)
        ON DELETE CASCADE,

    FOREIGN KEY (role_id)
        REFERENCES roles(id)
        ON DELETE CASCADE
);

CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    subject_id UUID NOT NULL,
    resource TEXT NOT NULL,
    action TEXT NOT NULL,
    decision TEXT NOT NULL CHECK (decision IN ('ALLOW', 'DENY')),
    reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    FOREIGN KEY (tenant_id)
        REFERENCES tenants(id)
        ON DELETE CASCADE
);

CREATE TABLE outbox (
    id BIGSERIAL PRIMARY KEY,
    event_type TEXT NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMPTZ
);

-- indexes
CREATE INDEX idx_roles_tenant
    ON roles (tenant_id);

CREATE INDEX idx_permissions_tenant
    ON permissions (tenant_id);

CREATE INDEX idx_subject_roles_subject
    ON subject_roles (tenant_id, subject_id);

CREATE INDEX idx_role_permissions_role
    ON role_permissions (tenant_id, role_id);

CREATE INDEX idx_role_inheritance_role
    ON role_inheritance (tenant_id, role_id);

CREATE INDEX idx_audit_logs_tenant
    ON audit_logs (tenant_id);

CREATE INDEX idx_outbox_processed
    ON outbox (processed_at);
