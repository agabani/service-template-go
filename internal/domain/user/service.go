package user

//go:generate go run go.uber.org/mock/mockgen -destination=mocks/mock_service.go -package=mocks github.com/agabani/service-template-go/internal/domain/user Service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/agabani/service-template-go/internal/domain"
)

type Service interface {
	Create(ctx context.Context, input CreateInput) (*User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	List(ctx context.Context, input domain.PageInput) (domain.Page[User], error)
	Update(ctx context.Context, id uuid.UUID, input UpdateInput) (*User, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) Create(ctx context.Context, input CreateInput) (*User, error) {
	if input.Email == "" {
		return nil, fmt.Errorf("%w: email is required", domain.ErrValidation)
	}
	if input.Name == "" {
		return nil, fmt.Errorf("%w: name is required", domain.ErrValidation)
	}
	return s.repo.Create(ctx, input)
}

func (s *service) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *service) List(ctx context.Context, input domain.PageInput) (domain.Page[User], error) {
	if input.Size <= 0 {
		input.Size = domain.PageSizeDefault
	}
	if input.Size > domain.PageSizeMax {
		input.Size = domain.PageSizeMax
	}
	return s.repo.List(ctx, input)
}

func (s *service) Update(ctx context.Context, id uuid.UUID, input UpdateInput) (*User, error) {
	return s.repo.Update(ctx, id, input)
}

func (s *service) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}
