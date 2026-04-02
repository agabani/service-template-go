//go:build integration

package integration_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/agabani/service-template-go/internal/adapters/http/handler"
	"github.com/agabani/service-template-go/internal/adapters/http/jsonapi"
)

func TestAccountCRUD(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()
	client := srv.Client()

	userID := createTestUser(t, client, srv.URL, "owner@example.com", "Owner")

	t.Run("create account", func(t *testing.T) {
		body := jsonapi.Document[map[string]any]{
			Data: &jsonapi.Resource[map[string]any]{
				Type: "accounts",
				Attributes: map[string]any{
					"name":     "Savings",
					"currency": "USD",
				},
			},
		}
		b, _ := json.Marshal(body)
		resp, err := client.Post(
			fmt.Sprintf("%s/users/%s/accounts", srv.URL, userID),
			"application/vnd.api+json",
			bytes.NewReader(b),
		)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var doc jsonapi.Document[handler.AccountAttributes]
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))
		require.NotNil(t, doc.Data)
		assert.Equal(t, "accounts", doc.Data.Type)
		assert.Equal(t, "Savings", doc.Data.Attributes.Name)
		assert.Equal(t, "USD", doc.Data.Attributes.Currency)
		assert.Equal(t, int64(0), doc.Data.Attributes.Balance)
		assert.Equal(t, userID, doc.Data.Attributes.UserID)
	})

	t.Run("list accounts for user", func(t *testing.T) {
		resp, err := client.Get(fmt.Sprintf("%s/users/%s/accounts", srv.URL, userID))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var doc jsonapi.DocumentList[handler.AccountAttributes]
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))
		assert.NotEmpty(t, doc.Data)
	})

	t.Run("get account by id", func(t *testing.T) {
		accountID := createTestAccount(t, client, srv.URL, userID, "Checking", "EUR")

		resp, err := client.Get(fmt.Sprintf("%s/accounts/%s", srv.URL, accountID))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var doc jsonapi.Document[handler.AccountAttributes]
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))
		assert.Equal(t, accountID, doc.Data.ID)
	})

	t.Run("update account", func(t *testing.T) {
		accountID := createTestAccount(t, client, srv.URL, userID, "Old Name", "GBP")

		body := jsonapi.Document[map[string]any]{
			Data: &jsonapi.Resource[map[string]any]{
				Type:       "accounts",
				Attributes: map[string]any{"name": "New Name"},
			},
		}
		b, _ := json.Marshal(body)
		req, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("%s/accounts/%s", srv.URL, accountID), bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/vnd.api+json")

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var doc jsonapi.Document[handler.AccountAttributes]
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))
		assert.Equal(t, "New Name", doc.Data.Attributes.Name)
	})

	t.Run("delete account", func(t *testing.T) {
		accountID := createTestAccount(t, client, srv.URL, userID, "To Delete", "JPY")

		req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/accounts/%s", srv.URL, accountID), nil)
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	})

	t.Run("invalid currency returns 400", func(t *testing.T) {
		body := jsonapi.Document[map[string]any]{
			Data: &jsonapi.Resource[map[string]any]{
				Type: "accounts",
				Attributes: map[string]any{
					"name":     "Bad",
					"currency": "US",
				},
			},
		}
		b, _ := json.Marshal(body)
		resp, err := client.Post(
			fmt.Sprintf("%s/users/%s/accounts", srv.URL, userID),
			"application/vnd.api+json",
			bytes.NewReader(b),
		)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func createTestAccount(t *testing.T, client *http.Client, base, userID, name, currency string) string {
	t.Helper()
	body := jsonapi.Document[map[string]any]{
		Data: &jsonapi.Resource[map[string]any]{
			Type: "accounts",
			Attributes: map[string]any{
				"name":     name,
				"currency": currency,
			},
		},
	}
	b, _ := json.Marshal(body)
	resp, err := client.Post(
		fmt.Sprintf("%s/users/%s/accounts", base, userID),
		"application/vnd.api+json",
		bytes.NewReader(b),
	)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var doc jsonapi.Document[handler.AccountAttributes]
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))
	return doc.Data.ID
}
