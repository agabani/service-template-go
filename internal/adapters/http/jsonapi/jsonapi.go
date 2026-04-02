package jsonapi

const ContentType = "application/vnd.api+json"

// Document is a JSON:API document containing a single resource.
type Document[T any] struct {
	Data   *Resource[T] `json:"data,omitempty"`
	Errors []Error      `json:"errors,omitempty"`
	Meta   any          `json:"meta,omitempty"`
}

// DocumentList is a JSON:API document containing multiple resources.
type DocumentList[T any] struct {
	Data  []Resource[T]      `json:"data"`
	Links *DocumentListLinks `json:"links,omitempty"`
}

// DocumentListLinks holds the pagination links for a collection document.
type DocumentListLinks struct {
	Next *string `json:"next"`
	Prev *string `json:"prev"`
}

// Resource is a JSON:API resource object.
type Resource[T any] struct {
	ID            string                  `json:"id"`
	Type          string                  `json:"type"`
	Attributes    T                       `json:"attributes"`
	Relationships map[string]Relationship `json:"relationships,omitempty"`
	Links         *Links                  `json:"links,omitempty"`
}

// Relationship is a JSON:API relationship object.
type Relationship struct {
	Data  *RelationshipData `json:"data,omitempty"`
	Links *Links            `json:"links,omitempty"`
}

// RelationshipData is the resource linkage for a relationship.
type RelationshipData struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

// Links holds JSON:API link objects.
type Links struct {
	Self    string `json:"self,omitempty"`
	Related string `json:"related,omitempty"`
}

// Error is a JSON:API error object.
type Error struct {
	Status string       `json:"status"`
	Title  string       `json:"title"`
	Detail string       `json:"detail,omitempty"`
	Source *ErrorSource `json:"source,omitempty"`
}

// ErrorSource identifies the source of a JSON:API error.
type ErrorSource struct {
	Pointer   string `json:"pointer,omitempty"`
	Parameter string `json:"parameter,omitempty"`
}

// ErrorDocument is a JSON:API document containing only errors.
type ErrorDocument struct {
	Errors []Error `json:"errors"`
}

func NewDocument[T any](id, resourceType string, attributes T) Document[T] {
	return Document[T]{
		Data: &Resource[T]{
			ID:         id,
			Type:       resourceType,
			Attributes: attributes,
		},
	}
}

func NewDocumentList[T any](resources []Resource[T]) DocumentList[T] {
	if resources == nil {
		resources = []Resource[T]{}
	}
	return DocumentList[T]{Data: resources}
}

func NewResource[T any](id, resourceType string, attributes T) Resource[T] {
	return Resource[T]{
		ID:         id,
		Type:       resourceType,
		Attributes: attributes,
	}
}
