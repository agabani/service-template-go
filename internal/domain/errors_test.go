package domain_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/agabani/service-template-go/internal/domain"
)

func TestErrors_areDistinct(t *testing.T) {
	assert.NotNil(t, domain.ErrNotFound)
	assert.NotNil(t, domain.ErrConflict)
	assert.NotNil(t, domain.ErrValidation)
	assert.False(t, errors.Is(domain.ErrNotFound, domain.ErrConflict))
	assert.False(t, errors.Is(domain.ErrNotFound, domain.ErrValidation))
	assert.False(t, errors.Is(domain.ErrConflict, domain.ErrValidation))
}
