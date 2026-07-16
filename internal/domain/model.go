// Package domain
package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Tenant struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	Name        string     `json:"name" db:"name"`
	RbacVersion int64      `json:"rbac_version" db:"rbac_version"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`
}

type Role struct {
	ID          uuid.UUID `json:"id" db:"id"`
	TenantID    uuid.UUID `json:"tenant_id" db:"tenant_id"`
	Name        string    `json:"name" db:"name"`
	Description *string   `json:"description,omitempty" db:"description"` 
	IsDisabled  bool      `json:"is_disabled" db:"is_disabled"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

type Permission struct {
	ID          uuid.UUID `json:"id" db:"id"`
	TenantID    uuid.UUID `json:"tenant_id" db:"tenant_id"`
	Resource    string    `json:"resource" db:"resource"`
	Action      string    `json:"action" db:"action"`
	Description *string   `json:"description,omitempty" db:"description"`
}

type RoleInheritance struct {
	TenantID           uuid.UUID `json:"tenant_id" db:"tenant_id"`
	RoleID             uuid.UUID `json:"role_id" db:"role_id"`
	InheritsFromRoleID uuid.UUID `json:"inherits_from_role_id" db:"inherits_from_role_id"`
}

type RolePermission struct {
	TenantID     uuid.UUID `json:"tenant_id" db:"tenant_id"`
	RoleID       uuid.UUID `json:"role_id" db:"role_id"`
	PermissionID uuid.UUID `json:"permission_id" db:"permission_id"`
}

type SubjectRole struct {
	TenantID  uuid.UUID `json:"tenant_id" db:"tenant_id"`
	SubjectID uuid.UUID `json:"subject_id" db:"subject_id"`
	RoleID    uuid.UUID `json:"role_id" db:"role_id"`
}

type AuditLog struct {
	ID        uuid.UUID `json:"id" db:"id"`
	TenantID  uuid.UUID `json:"tenant_id" db:"tenant_id"`
	SubjectID uuid.UUID `json:"subject_id" db:"subject_id"`
	Resource  string    `json:"resource" db:"resource"`
	Action    string    `json:"action" db:"action"`
	Decision  string    `json:"decision" db:"decision"` // 'ALLOW' or 'DENY'
	Reason    *string   `json:"reason,omitempty" db:"reason"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type Outbox struct {
	ID          int64           `json:"id" db:"id"`
	EventType   string          `json:"event_type" db:"event_type"`
	Payload     json.RawMessage `json:"payload" db:"payload"` // json.RawMessage handles database JSONB cleanly
	CreatedAt   time.Time       `json:"created_at" db:"created_at"`
	ProcessedAt *time.Time      `json:"processed_at,omitempty" db:"processed_at"`
}
