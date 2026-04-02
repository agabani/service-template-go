package handler_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/agabani/service-template-go/internal/adapters/http/handler"
	"github.com/agabani/service-template-go/internal/domain"
	usermocks "github.com/agabani/service-template-go/internal/domain/user/mocks"
)

func TestWriteError_conflict(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc := usermocks.NewMockService(ctrl)
	svc.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil, domain.ErrConflict)

	body := `{"data":{"type":"users","attributes":{}}}`
	rec := httptest.NewRecorder()
	handler.NewUserHandler(svc).Create(rec, httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(body)))

	assert.Equal(t, http.StatusConflict, rec.Code)
}

func TestWriteError_internalError(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc := usermocks.NewMockService(ctrl)
	svc.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil, errors.New("unexpected"))

	body := `{"data":{"type":"users","attributes":{}}}`
	rec := httptest.NewRecorder()
	handler.NewUserHandler(svc).Create(rec, httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(body)))

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}
