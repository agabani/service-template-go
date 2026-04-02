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

func TestUserCRUD(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()
	client := srv.Client()
	base := srv.URL

	t.Run("create user", func(t *testing.T) {
		body := jsonapi.Document[map[string]any]{
			Data: &jsonapi.Resource[map[string]any]{
				Type: "users",
				Attributes: map[string]any{
					"email": "alice@example.com",
					"name":  "Alice",
				},
			},
		}
		b, _ := json.Marshal(body)
		resp, err := client.Post(base+"/users", "application/vnd.api+json", bytes.NewReader(b))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var doc jsonapi.Document[handler.UserAttributes]
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))
		require.NotNil(t, doc.Data)
		assert.Equal(t, "users", doc.Data.Type)
		assert.Equal(t, "alice@example.com", doc.Data.Attributes.Email)
		assert.Equal(t, "Alice", doc.Data.Attributes.Name)
	})

	t.Run("list users", func(t *testing.T) {
		resp, err := client.Get(base + "/users")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var doc jsonapi.DocumentList[handler.UserAttributes]
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))
		assert.NotEmpty(t, doc.Data)
	})

	t.Run("get user by id", func(t *testing.T) {
		id := createTestUser(t, client, base, "bob@example.com", "Bob")

		resp, err := client.Get(fmt.Sprintf("%s/users/%s", base, id))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var doc jsonapi.Document[handler.UserAttributes]
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))
		assert.Equal(t, id, doc.Data.ID)
	})

	t.Run("update user", func(t *testing.T) {
		id := createTestUser(t, client, base, "charlie@example.com", "Charlie")
		newName := "Charlie Updated"

		body := jsonapi.Document[map[string]any]{
			Data: &jsonapi.Resource[map[string]any]{
				Type:       "users",
				Attributes: map[string]any{"name": newName},
			},
		}
		b, _ := json.Marshal(body)
		req, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("%s/users/%s", base, id), bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/vnd.api+json")

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var doc jsonapi.Document[handler.UserAttributes]
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))
		assert.Equal(t, newName, doc.Data.Attributes.Name)
	})

	t.Run("delete user", func(t *testing.T) {
		id := createTestUser(t, client, base, "dave@example.com", "Dave")

		req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/users/%s", base, id), nil)
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNoContent, resp.StatusCode)

		resp2, err := client.Get(fmt.Sprintf("%s/users/%s", base, id))
		require.NoError(t, err)
		defer resp2.Body.Close()
		assert.Equal(t, http.StatusNotFound, resp2.StatusCode)
	})

	t.Run("get non-existent user returns 404", func(t *testing.T) {
		resp, err := client.Get(base + "/users/00000000-0000-0000-0000-000000000000")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}

func createTestUser(t *testing.T, client *http.Client, base, email, name string) string {
	t.Helper()
	body := jsonapi.Document[map[string]any]{
		Data: &jsonapi.Resource[map[string]any]{
			Type: "users",
			Attributes: map[string]any{
				"email": email,
				"name":  name,
			},
		},
	}
	b, _ := json.Marshal(body)
	resp, err := client.Post(base+"/users", "application/vnd.api+json", bytes.NewReader(b))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var doc jsonapi.Document[handler.UserAttributes]
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))
	return doc.Data.ID
}

func TestHealthEndpoints(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	t.Run("liveness", func(t *testing.T) {
		resp, err := srv.Client().Get(srv.URL + "/health/live")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("readiness", func(t *testing.T) {
		resp, err := srv.Client().Get(srv.URL + "/health/ready")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

func TestUserValidation(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()
	client := srv.Client()

	t.Run("missing email returns 400", func(t *testing.T) {
		body := jsonapi.Document[map[string]any]{
			Data: &jsonapi.Resource[map[string]any]{
				Type:       "users",
				Attributes: map[string]any{"name": "No Email"},
			},
		}
		b, _ := json.Marshal(body)
		resp, err := client.Post(srv.URL+"/users", "application/vnd.api+json", bytes.NewReader(b))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("duplicate email returns 409", func(t *testing.T) {
		createTestUser(t, client, srv.URL, "unique@example.com", "Unique")

		body := jsonapi.Document[map[string]any]{
			Data: &jsonapi.Resource[map[string]any]{
				Type: "users",
				Attributes: map[string]any{
					"email": "unique@example.com",
					"name":  "Duplicate",
				},
			},
		}
		b, _ := json.Marshal(body)
		resp, err := client.Post(srv.URL+"/users", "application/vnd.api+json", bytes.NewReader(b))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusConflict, resp.StatusCode)
	})
}
