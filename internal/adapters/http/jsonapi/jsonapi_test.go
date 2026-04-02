package jsonapi_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/agabani/service-template-go/internal/adapters/http/jsonapi"
)

func TestNewDocument(t *testing.T) {
	type attrs struct{ Val string }
	doc := jsonapi.NewDocument("1", "things", attrs{Val: "x"})
	require.NotNil(t, doc.Data)
	assert.Equal(t, "1", doc.Data.ID)
	assert.Equal(t, "things", doc.Data.Type)
	assert.Equal(t, attrs{Val: "x"}, doc.Data.Attributes)
}

func TestNewDocumentList_nilBecomesEmpty(t *testing.T) {
	type attrs struct{}
	doc := jsonapi.NewDocumentList[attrs](nil)
	assert.NotNil(t, doc.Data)
	assert.Empty(t, doc.Data)
}

func TestNewDocumentList_preservesSlice(t *testing.T) {
	type attrs struct{ N int }
	resources := []jsonapi.Resource[attrs]{
		jsonapi.NewResource("1", "things", attrs{N: 1}),
		jsonapi.NewResource("2", "things", attrs{N: 2}),
	}
	doc := jsonapi.NewDocumentList(resources)
	assert.Len(t, doc.Data, 2)
}

func TestNewResource(t *testing.T) {
	type attrs struct{ Val int }
	r := jsonapi.NewResource("42", "items", attrs{Val: 7})
	assert.Equal(t, "42", r.ID)
	assert.Equal(t, "items", r.Type)
	assert.Equal(t, attrs{Val: 7}, r.Attributes)
}

func TestDocumentListLinks_JSON_bothLinks(t *testing.T) {
	next := "http://example.com/users?page%5Bafter%5D=abc"
	prev := "http://example.com/users?page%5Bbefore%5D=xyz"
	links := jsonapi.DocumentListLinks{Next: &next, Prev: &prev}
	b, err := json.Marshal(links)
	require.NoError(t, err)
	assert.JSONEq(t, `{"next":"http://example.com/users?page%5Bafter%5D=abc","prev":"http://example.com/users?page%5Bbefore%5D=xyz"}`, string(b))
}

func TestDocumentListLinks_JSON_noLinks(t *testing.T) {
	links := jsonapi.DocumentListLinks{}
	b, err := json.Marshal(links)
	require.NoError(t, err)
	assert.JSONEq(t, `{"next":null,"prev":null}`, string(b))
}

func TestErrorDocument_JSON(t *testing.T) {
	doc := jsonapi.ErrorDocument{
		Errors: []jsonapi.Error{
			{Status: "404", Title: "Not Found", Detail: "missing"},
		},
	}
	b, err := json.Marshal(doc)
	require.NoError(t, err)
	assert.JSONEq(t, `{"errors":[{"status":"404","title":"Not Found","detail":"missing"}]}`, string(b))
}
