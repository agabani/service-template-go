package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/agabani/service-template-go/internal/adapters/http/handler"
	"github.com/agabani/service-template-go/internal/domain"
	"github.com/agabani/service-template-go/internal/domain/account"
	accountmocks "github.com/agabani/service-template-go/internal/domain/account/mocks"
)

func TestAccountHandler_ListByUser_success(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc := accountmocks.NewMockService(ctrl)
	userID := uuid.New()
	svc.EXPECT().ListByUserID(gomock.Any(), userID, gomock.Any()).Return(domain.Page[account.Account]{
		Items: []*account.Account{{ID: uuid.New(), UserID: userID, Name: "Savings", Currency: "USD"}},
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/users/"+userID.String()+"/accounts", nil)
	req.SetPathValue("userID", userID.String())
	rec := httptest.NewRecorder()
	handler.NewAccountHandler(svc).ListByUser(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	data, _ := body["data"].([]any)
	assert.Len(t, data, 1)
}

func TestAccountHandler_ListByUser_invalidUserID(t *testing.T) {
	ctrl := gomock.NewController(t)
	req := httptest.NewRequest(http.MethodGet, "/users/bad/accounts", nil)
	req.SetPathValue("userID", "bad")
	rec := httptest.NewRecorder()
	handler.NewAccountHandler(accountmocks.NewMockService(ctrl)).ListByUser(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAccountHandler_Create_success(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc := accountmocks.NewMockService(ctrl)
	userID := uuid.New()
	svc.EXPECT().Create(gomock.Any(), gomock.Any()).Return(&account.Account{
		ID: uuid.New(), UserID: userID, Name: "Savings", Currency: "USD",
	}, nil)

	body := `{"data":{"type":"accounts","attributes":{"name":"Savings","currency":"USD"}}}`
	req := httptest.NewRequest(http.MethodPost, "/users/"+userID.String()+"/accounts", strings.NewReader(body))
	req.SetPathValue("userID", userID.String())
	rec := httptest.NewRecorder()
	handler.NewAccountHandler(svc).Create(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)
}

func TestAccountHandler_Create_invalidUserID(t *testing.T) {
	ctrl := gomock.NewController(t)
	req := httptest.NewRequest(http.MethodPost, "/users/bad/accounts", strings.NewReader(`{}`))
	req.SetPathValue("userID", "bad")
	rec := httptest.NewRecorder()
	handler.NewAccountHandler(accountmocks.NewMockService(ctrl)).Create(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAccountHandler_Get_notFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc := accountmocks.NewMockService(ctrl)
	id := uuid.New()
	svc.EXPECT().GetByID(gomock.Any(), id).Return(nil, domain.ErrNotFound)

	req := httptest.NewRequest(http.MethodGet, "/accounts/"+id.String(), nil)
	req.SetPathValue("id", id.String())
	rec := httptest.NewRecorder()
	handler.NewAccountHandler(svc).Get(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestAccountHandler_Update_success(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc := accountmocks.NewMockService(ctrl)
	id := uuid.New()
	svc.EXPECT().Update(gomock.Any(), id, gomock.Any()).Return(&account.Account{ID: id, Name: "New"}, nil)

	body := `{"data":{"type":"accounts","attributes":{"name":"New"}}}`
	req := httptest.NewRequest(http.MethodPatch, "/accounts/"+id.String(), strings.NewReader(body))
	req.SetPathValue("id", id.String())
	rec := httptest.NewRecorder()
	handler.NewAccountHandler(svc).Update(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestAccountHandler_Delete_notFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc := accountmocks.NewMockService(ctrl)
	id := uuid.New()
	svc.EXPECT().Delete(gomock.Any(), id).Return(domain.ErrNotFound)

	req := httptest.NewRequest(http.MethodDelete, "/accounts/"+id.String(), nil)
	req.SetPathValue("id", id.String())
	rec := httptest.NewRecorder()
	handler.NewAccountHandler(svc).Delete(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}
