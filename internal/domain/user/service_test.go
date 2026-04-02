package user_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/agabani/service-template-go/internal/domain"
	"github.com/agabani/service-template-go/internal/domain/user"
	"github.com/agabani/service-template-go/internal/domain/user/mocks"
)

func TestService_Create(t *testing.T) {
	t.Run("succeeds with valid input", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockRepository(ctrl)
		want := &user.User{ID: uuid.New(), Email: "a@b.com", Name: "A"}
		repo.EXPECT().Create(gomock.Any(), user.CreateInput{Email: "a@b.com", Name: "A"}).Return(want, nil)

		svc := user.NewService(repo)
		got, err := svc.Create(context.Background(), user.CreateInput{Email: "a@b.com", Name: "A"})
		require.NoError(t, err)
		assert.Equal(t, want, got)
	})

	t.Run("returns validation error when email is empty", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockRepository(ctrl)

		svc := user.NewService(repo)
		_, err := svc.Create(context.Background(), user.CreateInput{Name: "A"})
		assert.True(t, errors.Is(err, domain.ErrValidation))
	})

	t.Run("returns validation error when name is empty", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockRepository(ctrl)

		svc := user.NewService(repo)
		_, err := svc.Create(context.Background(), user.CreateInput{Email: "a@b.com"})
		assert.True(t, errors.Is(err, domain.ErrValidation))
	})
}

func TestService_List_defaultsSize(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockRepository(ctrl)
	repo.EXPECT().List(gomock.Any(), domain.PageInput{Size: domain.PageSizeDefault}).Return(domain.Page[user.User]{}, nil)

	svc := user.NewService(repo)
	_, err := svc.List(context.Background(), domain.PageInput{})
	require.NoError(t, err)
}

func TestService_List_clampsSize(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockRepository(ctrl)
	repo.EXPECT().List(gomock.Any(), domain.PageInput{Size: domain.PageSizeMax}).Return(domain.Page[user.User]{}, nil)

	svc := user.NewService(repo)
	_, err := svc.List(context.Background(), domain.PageInput{Size: 9999})
	require.NoError(t, err)
}

func TestService_GetByID(t *testing.T) {
	t.Run("returns not found when repo returns not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockRepository(ctrl)
		repo.EXPECT().GetByID(gomock.Any(), gomock.Any()).Return(nil, domain.ErrNotFound)

		svc := user.NewService(repo)
		_, err := svc.GetByID(context.Background(), uuid.New())
		assert.True(t, errors.Is(err, domain.ErrNotFound))
	})
}
