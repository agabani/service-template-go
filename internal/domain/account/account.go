// Package account implements the Account domain: entity, repository port
// (interface), and service. Validation and business rules belong here. This
// package must not import adapters or config — only the standard library and
// pure Go libraries.
package account

import (
	"time"

	"github.com/google/uuid"
)

type Account struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Name      string
	Balance   int64
	Currency  string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type CreateInput struct {
	UserID   uuid.UUID
	Name     string
	Currency string
}

type UpdateInput struct {
	Name *string
}
