package account

//go:generate go run go.uber.org/mock/mockgen -destination=mocks/mock_repository.go -package=mocks github.com/agabani/service-template-go/internal/domain/account Repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/agabani/service-template-go/internal/domain"
)

type Repository interface {
	Create(ctx context.Context, input CreateInput) (*Account, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Account, error)
	ListByUserID(ctx context.Context, userID uuid.UUID, input domain.PageInput) (domain.Page[Account], error)
	Update(ctx context.Context, id uuid.UUID, input UpdateInput) (*Account, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
