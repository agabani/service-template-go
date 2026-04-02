package user

//go:generate go run go.uber.org/mock/mockgen -destination=mocks/mock_repository.go -package=mocks github.com/agabani/service-template-go/internal/domain/user Repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/agabani/service-template-go/internal/domain"
)

type Repository interface {
	Create(ctx context.Context, input CreateInput) (*User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	List(ctx context.Context, input domain.PageInput) (domain.Page[User], error)
	Update(ctx context.Context, id uuid.UUID, input UpdateInput) (*User, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
