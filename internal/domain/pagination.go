package domain

const (
	PageSizeDefault = 20
	PageSizeMax     = 100
)

// PageInput carries cursor-based pagination parameters for list operations.
type PageInput struct {
	Size   int
	After  *PageCursor // fetch items after this cursor (next page)
	Before *PageCursor // fetch items before this cursor (prev page)
}

// PageCursor is the keyset position used to navigate between pages.
// It encodes the internal BIGSERIAL id of a row at a page boundary.
type PageCursor struct {
	ID int64
}

// Page is the result of a paginated list operation.
type Page[T any] struct {
	Items []*T
	Next  *PageCursor
	Prev  *PageCursor
}
