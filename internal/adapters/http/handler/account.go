package handler

import (
	"net/http"
	"time"

	"github.com/agabani/service-template-go/internal/adapters/http/jsonapi"
	"github.com/agabani/service-template-go/internal/domain/account"
)

type AccountAttributes struct {
	UserID    string    `json:"user_id"`
	Name      string    `json:"name"`
	Balance   int64     `json:"balance"`
	Currency  string    `json:"currency"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateAccountAttributes struct {
	Name     string `json:"name"`
	Currency string `json:"currency"`
}

type UpdateAccountAttributes struct {
	Name *string `json:"name"`
}

type AccountHandler struct {
	service account.Service
}

func NewAccountHandler(svc account.Service) *AccountHandler {
	return &AccountHandler{service: svc}
}

func (h *AccountHandler) ListByUser(w http.ResponseWriter, r *http.Request) {
	userID, ok := pathUUID(w, r, "userID", "invalid user_id")
	if !ok {
		return
	}
	pageInput, ok := parsePage(w, r)
	if !ok {
		return
	}
	page, err := h.service.ListByUserID(r.Context(), userID, pageInput)
	if err != nil {
		writeError(w, r, err)
		return
	}
	resources := make([]jsonapi.Resource[AccountAttributes], 0, len(page.Items))
	for _, a := range page.Items {
		resources = append(resources, accountToResource(a))
	}
	doc := jsonapi.NewDocumentList(resources)
	doc.Links = &jsonapi.DocumentListLinks{
		Next: paginationURL(r, "page[after]", page.Next),
		Prev: paginationURL(r, "page[before]", page.Prev),
	}
	writeJSON(w, r, http.StatusOK, doc)
}

func (h *AccountHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := pathUUID(w, r, "userID", "invalid user_id")
	if !ok {
		return
	}
	data, ok := decodeBody[CreateAccountAttributes](w, r)
	if !ok {
		return
	}
	a, err := h.service.Create(r.Context(), account.CreateInput{
		UserID:   userID,
		Name:     data.Attributes.Name,
		Currency: data.Attributes.Currency,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, http.StatusCreated, accountDocument(a))
}

func (h *AccountHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := pathUUID(w, r, "id", "invalid id")
	if !ok {
		return
	}
	a, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, http.StatusOK, accountDocument(a))
}

func (h *AccountHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := pathUUID(w, r, "id", "invalid id")
	if !ok {
		return
	}
	data, ok := decodeBody[UpdateAccountAttributes](w, r)
	if !ok {
		return
	}
	a, err := h.service.Update(r.Context(), id, account.UpdateInput{
		Name: data.Attributes.Name,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, http.StatusOK, accountDocument(a))
}

func (h *AccountHandler) Delete(w http.ResponseWriter, r *http.Request) {
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

func accountDocument(a *account.Account) jsonapi.Document[AccountAttributes] {
	return jsonapi.NewDocument(a.ID.String(), "accounts", accountAttributes(a))
}

func accountToResource(a *account.Account) jsonapi.Resource[AccountAttributes] {
	return jsonapi.NewResource(a.ID.String(), "accounts", accountAttributes(a))
}

func accountAttributes(a *account.Account) AccountAttributes {
	return AccountAttributes{
		UserID:    a.UserID.String(),
		Name:      a.Name,
		Balance:   a.Balance,
		Currency:  a.Currency,
		CreatedAt: a.CreatedAt,
		UpdatedAt: a.UpdatedAt,
	}
}
