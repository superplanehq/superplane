package database

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRedactSQLParams(t *testing.T) {
	now := time.Now()
	params := redactSQLParams(
		"secret-token",
		[]byte("binary-secret"),
		42,
		true,
		now,
		&now,
		nil,
	)

	assert.Equal(t, "?", params[0])
	assert.Equal(t, "?", params[1])
	assert.Equal(t, 42, params[2])
	assert.Equal(t, true, params[3])
	assert.Equal(t, "?", params[4])
	assert.Equal(t, "?", params[5])
	assert.Nil(t, params[6])
}

func TestSanitizeSQLStatementRedactsStringLiterals(t *testing.T) {
	sql := `SELECT * FROM "secrets" WHERE "secrets"."value" = 'super-secret' AND "secrets"."name" = 'O''Reilly'`
	want := `SELECT * FROM "secrets" WHERE "secrets"."value" = '?' AND "secrets"."name" = '?'`

	assert.Equal(t, want, sanitizeSQLStatement(sql))
}

func TestSanitizeSQLStatementRedactsEscapesAndDollarQuotes(t *testing.T) {
	sql := `SELECT E'\\xdeadbeef' AS payload, $$multi
line secret$$ AS body`
	want := `SELECT '?' AS payload, '?' AS body`

	assert.Equal(t, want, sanitizeSQLStatement(sql))
}

func TestSanitizeSQLStatementPreservesAlreadyRedactedLiterals(t *testing.T) {
	sql := `SELECT * FROM users WHERE email = '?' AND id = 42`

	assert.Equal(t, sql, sanitizeSQLStatement(sql))
}

func TestSanitizeSQLStatementPreservesStructureWithoutLiterals(t *testing.T) {
	sql := `SELECT * FROM users WHERE id = $1 AND deleted_at IS NULL`

	assert.Equal(t, sql, sanitizeSQLStatement(sql))
}
