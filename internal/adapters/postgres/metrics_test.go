package postgres_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/agabani/service-template-go/internal/adapters/postgres"
)

// TestRegisterPoolMetrics_returnsNoError verifies that metric registration
// succeeds with the default noop meter provider used in unit test environments.
// The pool callback is never invoked by the noop provider.
func TestRegisterPoolMetrics_returnsNoError(t *testing.T) {
	cleanup, err := postgres.RegisterPoolMetrics(nil)
	require.NoError(t, err)
	require.NotNil(t, cleanup)
	assert.NotPanics(t, cleanup)
}
