package account_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/agabani/service-template-go/internal/domain"
	"github.com/agabani/service-template-go/internal/domain/account"
	"github.com/agabani/service-template-go/internal/domain/account/mocks"
)

func TestService_ListByUserID_defaultsSize(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockRepository(ctrl)
	repo.EXPECT().ListByUserID(gomock.Any(), gomock.Any(), domain.PageInput{Size: domain.PageSizeDefault}).Return(domain.Page[account.Account]{}, nil)

	svc := account.NewService(repo)
	_, err := svc.ListByUserID(context.Background(), uuid.New(), domain.PageInput{})
	require.NoError(t, err)
}

func TestService_ListByUserID_clampsSize(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockRepository(ctrl)
	repo.EXPECT().ListByUserID(gomock.Any(), gomock.Any(), domain.PageInput{Size: domain.PageSizeMax}).Return(domain.Page[account.Account]{}, nil)

	svc := account.NewService(repo)
	_, err := svc.ListByUserID(context.Background(), uuid.New(), domain.PageInput{Size: 9999})
	require.NoError(t, err)
}

func TestService_Create(t *testing.T) {
	t.Run("succeeds with valid input", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockRepository(ctrl)
		want := &account.Account{ID: uuid.New(), Name: "Savings", Currency: "USD"}
		repo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(want, nil)

		svc := account.NewService(repo)
		got, err := svc.Create(context.Background(), account.CreateInput{
			UserID:   uuid.New(),
			Name:     "Savings",
			Currency: "USD",
		})
		require.NoError(t, err)
		assert.Equal(t, want, got)
	})

	t.Run("returns validation error when name is empty", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockRepository(ctrl)

		svc := account.NewService(repo)
		_, err := svc.Create(context.Background(), account.CreateInput{Currency: "USD"})
		assert.True(t, errors.Is(err, domain.ErrValidation))
	})

	t.Run("returns validation error when currency is invalid length", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockRepository(ctrl)

		svc := account.NewService(repo)
		_, err := svc.Create(context.Background(), account.CreateInput{
			Name:     "Account",
			Currency: "US",
		})
		assert.True(t, errors.Is(err, domain.ErrValidation))
	})
}
