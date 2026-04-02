package account

//go:generate go run go.uber.org/mock/mockgen -destination=mocks/mock_service.go -package=mocks github.com/agabani/service-template-go/internal/domain/account Service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/agabani/service-template-go/internal/domain"
)

type Service interface {
	Create(ctx context.Context, input CreateInput) (*Account, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Account, error)
	ListByUserID(ctx context.Context, userID uuid.UUID, input domain.PageInput) (domain.Page[Account], error)
	Update(ctx context.Context, id uuid.UUID, input UpdateInput) (*Account, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) Create(ctx context.Context, input CreateInput) (*Account, error) {
	if input.Name == "" {
		return nil, fmt.Errorf("%w: name is required", domain.ErrValidation)
	}
	if input.Currency == "" {
		return nil, fmt.Errorf("%w: currency is required", domain.ErrValidation)
	}
	if len(input.Currency) != 3 {
		return nil, fmt.Errorf("%w: currency must be a 3-letter ISO 4217 code", domain.ErrValidation)
	}
	return s.repo.Create(ctx, input)
}

func (s *service) GetByID(ctx context.Context, id uuid.UUID) (*Account, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *service) ListByUserID(ctx context.Context, userID uuid.UUID, input domain.PageInput) (domain.Page[Account], error) {
	if input.Size <= 0 {
		input.Size = domain.PageSizeDefault
	}
	if input.Size > domain.PageSizeMax {
		input.Size = domain.PageSizeMax
	}
	return s.repo.ListByUserID(ctx, userID, input)
}

func (s *service) Update(ctx context.Context, id uuid.UUID, input UpdateInput) (*Account, error) {
	return s.repo.Update(ctx, id, input)
}

func (s *service) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}
