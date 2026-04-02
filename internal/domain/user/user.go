// Package user implements the User domain: entity, repository port (interface),
// and service. Validation and business rules belong here. This package must not
// import adapters or config — only the standard library and pure Go libraries.
package user

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID        uuid.UUID
	Email     string
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type CreateInput struct {
	Email string
	Name  string
}

type UpdateInput struct {
	Email *string
	Name  *string
}
