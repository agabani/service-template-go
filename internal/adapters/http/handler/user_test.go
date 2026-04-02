package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/agabani/service-template-go/internal/adapters/http/handler"
	"github.com/agabani/service-template-go/internal/domain"
	"github.com/agabani/service-template-go/internal/domain/user"
	usermocks "github.com/agabani/service-template-go/internal/domain/user/mocks"
)

func TestUserHandler_List_success(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc := usermocks.NewMockService(ctrl)
	id := uuid.New()
	svc.EXPECT().List(gomock.Any(), gomock.Any()).Return(domain.Page[user.User]{
		Items: []*user.User{{ID: id, Email: "a@b.com", Name: "Alice"}},
	}, nil)

	rec := httptest.NewRecorder()
	handler.NewUserHandler(svc).List(rec, httptest.NewRequest(http.MethodGet, "/users", nil))

	assert.Equal(t, http.StatusOK, rec.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	data, _ := body["data"].([]any)
	require.Len(t, data, 1)
	res := data[0].(map[string]any)
	assert.Equal(t, "users", res["type"])
}

func TestUserHandler_List_invalidPageSize(t *testing.T) {
	ctrl := gomock.NewController(t)
	rec := httptest.NewRecorder()
	handler.NewUserHandler(usermocks.NewMockService(ctrl)).List(
		rec, httptest.NewRequest(http.MethodGet, "/users?page[size]=0", nil),
	)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUserHandler_List_invalidAfterCursor(t *testing.T) {
	ctrl := gomock.NewController(t)
	rec := httptest.NewRecorder()
	handler.NewUserHandler(usermocks.NewMockService(ctrl)).List(
		rec, httptest.NewRequest(http.MethodGet, "/users?page[after]=notvalid!!", nil),
	)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUserHandler_List_afterAndBeforeMutuallyExclusive(t *testing.T) {
	ctrl := gomock.NewController(t)
	rec := httptest.NewRecorder()
	handler.NewUserHandler(usermocks.NewMockService(ctrl)).List(
		rec, httptest.NewRequest(http.MethodGet, "/users?page[after]=abc&page[before]=def", nil),
	)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUserHandler_Create_success(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc := usermocks.NewMockService(ctrl)
	created := &user.User{ID: uuid.New(), Email: "a@b.com", Name: "Alice", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	svc.EXPECT().Create(gomock.Any(), user.CreateInput{Email: "a@b.com", Name: "Alice"}).Return(created, nil)

	body := `{"data":{"type":"users","attributes":{"email":"a@b.com","name":"Alice"}}}`
	rec := httptest.NewRecorder()
	handler.NewUserHandler(svc).Create(rec, httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(body)))

	assert.Equal(t, http.StatusCreated, rec.Code)
}

func TestUserHandler_Create_invalidJSON(t *testing.T) {
	ctrl := gomock.NewController(t)
	rec := httptest.NewRecorder()
	handler.NewUserHandler(usermocks.NewMockService(ctrl)).Create(
		rec, httptest.NewRequest(http.MethodPost, "/users", strings.NewReader("not-json")),
	)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUserHandler_Create_nilData(t *testing.T) {
	ctrl := gomock.NewController(t)
	rec := httptest.NewRecorder()
	handler.NewUserHandler(usermocks.NewMockService(ctrl)).Create(
		rec, httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(`{}`)),
	)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUserHandler_Create_serviceError(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc := usermocks.NewMockService(ctrl)
	svc.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil, domain.ErrValidation)

	body := `{"data":{"type":"users","attributes":{}}}`
	rec := httptest.NewRecorder()
	handler.NewUserHandler(svc).Create(rec, httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(body)))

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestUserHandler_Get_success(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc := usermocks.NewMockService(ctrl)
	id := uuid.New()
	svc.EXPECT().GetByID(gomock.Any(), id).Return(&user.User{ID: id, Email: "a@b.com", Name: "A"}, nil)

	req := httptest.NewRequest(http.MethodGet, "/users/"+id.String(), nil)
	req.SetPathValue("id", id.String())
	rec := httptest.NewRecorder()
	handler.NewUserHandler(svc).Get(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestUserHandler_Get_invalidUUID(t *testing.T) {
	ctrl := gomock.NewController(t)
	req := httptest.NewRequest(http.MethodGet, "/users/bad", nil)
	req.SetPathValue("id", "bad")
	rec := httptest.NewRecorder()
	handler.NewUserHandler(usermocks.NewMockService(ctrl)).Get(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUserHandler_Get_notFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc := usermocks.NewMockService(ctrl)
	id := uuid.New()
	svc.EXPECT().GetByID(gomock.Any(), id).Return(nil, domain.ErrNotFound)

	req := httptest.NewRequest(http.MethodGet, "/users/"+id.String(), nil)
	req.SetPathValue("id", id.String())
	rec := httptest.NewRecorder()
	handler.NewUserHandler(svc).Get(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestUserHandler_Update_success(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc := usermocks.NewMockService(ctrl)
	id := uuid.New()
	name := "Bob"
	svc.EXPECT().Update(gomock.Any(), id, gomock.Any()).Return(&user.User{ID: id, Name: name}, nil)

	body := `{"data":{"type":"users","attributes":{"name":"Bob"}}}`
	req := httptest.NewRequest(http.MethodPatch, "/users/"+id.String(), strings.NewReader(body))
	req.SetPathValue("id", id.String())
	rec := httptest.NewRecorder()
	handler.NewUserHandler(svc).Update(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestUserHandler_Update_invalidUUID(t *testing.T) {
	ctrl := gomock.NewController(t)
	req := httptest.NewRequest(http.MethodPatch, "/users/bad", strings.NewReader(`{}`))
	req.SetPathValue("id", "bad")
	rec := httptest.NewRecorder()
	handler.NewUserHandler(usermocks.NewMockService(ctrl)).Update(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUserHandler_Delete_success(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc := usermocks.NewMockService(ctrl)
	id := uuid.New()
	svc.EXPECT().Delete(gomock.Any(), id).Return(nil)

	req := httptest.NewRequest(http.MethodDelete, "/users/"+id.String(), nil)
	req.SetPathValue("id", id.String())
	rec := httptest.NewRecorder()
	handler.NewUserHandler(svc).Delete(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestUserHandler_Delete_invalidUUID(t *testing.T) {
	ctrl := gomock.NewController(t)
	req := httptest.NewRequest(http.MethodDelete, "/users/bad", nil)
	req.SetPathValue("id", "bad")
	rec := httptest.NewRecorder()
	handler.NewUserHandler(usermocks.NewMockService(ctrl)).Delete(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
