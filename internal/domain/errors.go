// Package domain defines the sentinel errors shared across all domain
// sub-packages. It has no dependencies on adapters or infrastructure.
package domain

import "errors"

var (
	ErrNotFound   = errors.New("not found")
	ErrConflict   = errors.New("conflict")
	ErrValidation = errors.New("validation error")
)
