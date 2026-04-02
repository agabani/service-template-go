package handler

import (
	"net/http"
	"time"

	"github.com/agabani/service-template-go/internal/adapters/http/jsonapi"
	"github.com/agabani/service-template-go/internal/domain/user"
)

type UserAttributes struct {
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateUserAttributes struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

type UpdateUserAttributes struct {
	Email *string `json:"email"`
	Name  *string `json:"name"`
}

type UserHandler struct {
	service user.Service
}

func NewUserHandler(svc user.Service) *UserHandler {
	return &UserHandler{service: svc}
}

func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
	pageInput, ok := parsePage(w, r)
	if !ok {
		return
	}
	page, err := h.service.List(r.Context(), pageInput)
	if err != nil {
		writeError(w, r, err)
		return
	}
	resources := make([]jsonapi.Resource[UserAttributes], 0, len(page.Items))
	for _, u := range page.Items {
		resources = append(resources, userToResource(u))
	}
	doc := jsonapi.NewDocumentList(resources)
	doc.Links = &jsonapi.DocumentListLinks{
		Next: paginationURL(r, "page[after]", page.Next),
		Prev: paginationURL(r, "page[before]", page.Prev),
	}
	writeJSON(w, r, http.StatusOK, doc)
}

func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
	data, ok := decodeBody[CreateUserAttributes](w, r)
	if !ok {
		return
	}
	u, err := h.service.Create(r.Context(), user.CreateInput{
		Email: data.Attributes.Email,
		Name:  data.Attributes.Name,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, http.StatusCreated, userDocument(u))
}

func (h *UserHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := pathUUID(w, r, "id", "invalid id")
	if !ok {
		return
	}
	u, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, http.StatusOK, userDocument(u))
}

func (h *UserHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := pathUUID(w, r, "id", "invalid id")
	if !ok {
		return
	}
	data, ok := decodeBody[UpdateUserAttributes](w, r)
	if !ok {
		return
	}
	u, err := h.service.Update(r.Context(), id, user.UpdateInput{
		Email: data.Attributes.Email,
		Name:  data.Attributes.Name,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, http.StatusOK, userDocument(u))
}

func (h *UserHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := pathUUID(w, r, "id", "invalid id")
	if !ok {
		return
	}
	if err := h.service.Delete(r.Context(), id); err != nil {
		writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func userDocument(u *user.User) jsonapi.Document[UserAttributes] {
	return jsonapi.NewDocument(u.ID.String(), "users", userAttributes(u))
}

func userToResource(u *user.User) jsonapi.Resource[UserAttributes] {
	return jsonapi.NewResource(u.ID.String(), "users", userAttributes(u))
}

func userAttributes(u *user.User) UserAttributes {
	return UserAttributes{
		Email:     u.Email,
		Name:      u.Name,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}
