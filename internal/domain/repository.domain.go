package domain

import (
	"context"

	"github.com/google/uuid"
)

type RoleReader interface {
	GetRoleRepository(ctx context.Context, id uuid.UUID) (role Role, err error)
}

type RoleWriter interface {
	CreateRoleRepository(ctx context.Context, role Role) (id uuid.UUID, err error)
}

type RoleRepository interface {
	RoleReader
	RoleWriter
}
