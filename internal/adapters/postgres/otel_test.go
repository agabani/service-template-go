package postgres

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDbSpanName_insert(t *testing.T) {
	assert.Equal(t, "INSERT users", dbSpanName("INSERT INTO users (id) VALUES ($1)"))
}

func TestDbSpanName_update(t *testing.T) {
	assert.Equal(t, "UPDATE users", dbSpanName("UPDATE users SET name = $1 WHERE id = $2"))
}

func TestDbSpanName_delete(t *testing.T) {
	assert.Equal(t, "DELETE users", dbSpanName("DELETE FROM users WHERE id = $1"))
}

func TestDbSpanName_select(t *testing.T) {
	assert.Equal(t, "SELECT", dbSpanName("SELECT id, name FROM users"))
}

func TestDbSpanName_empty(t *testing.T) {
	assert.Equal(t, "db.query", dbSpanName(""))
}

func TestSqlVerb_returnsUppercaseVerb(t *testing.T) {
	assert.Equal(t, "SELECT", sqlVerb("select id from users"))
	assert.Equal(t, "INSERT", sqlVerb("INSERT INTO users"))
	assert.Equal(t, "UPDATE", sqlVerb("UPDATE users SET"))
	assert.Equal(t, "DELETE", sqlVerb("DELETE FROM users"))
}

func TestSqlVerb_singleWord(t *testing.T) {
	assert.Equal(t, "BEGIN", sqlVerb("BEGIN"))
}

func TestNewQueryTracer_returnsNoError(t *testing.T) {
	qt, err := newQueryTracer()
	require.NoError(t, err)
	require.NotNil(t, qt)
}
