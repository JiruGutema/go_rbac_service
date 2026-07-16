package domain

import (
	"context"

	"github.com/google/uuid"
)

type RoleRepository interface {
	CreateRole(ctx context.Context, name string) (id uuid.UUID)
	GetRole(ctx context.Context, id uuid.UUID) (role Role)
}


