package database

import (
	"fmt"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcessTestDatabaseName(t *testing.T) {
	assert.Equal(t, "superplane_12345_test", processTestDatabaseName(12345))
	assert.True(t, isTestDatabaseName(processTestDatabaseName(99)))
	assert.True(t, isProcessTestDatabaseName(processTestDatabaseName(99)))
}

func TestIsProcessTestDatabaseName(t *testing.T) {
	assert.True(t, isProcessTestDatabaseName("superplane_1_test"))
	assert.True(t, isProcessTestDatabaseName("superplane_44221_test"))
	assert.False(t, isProcessTestDatabaseName("superplane_test"))
	assert.False(t, isProcessTestDatabaseName("superplane_dev"))
	assert.False(t, isProcessTestDatabaseName("superplane_test_1"))
	assert.False(t, isProcessTestDatabaseName("superplane_abc_test"))
}

func TestShouldIsolateTestDatabase(t *testing.T) {
	t.Run("isolates the shared test database", func(t *testing.T) {
		t.Setenv(testProcessIsolateEnv, "")
		assert.True(t, shouldIsolateTestDatabase("superplane_test"))
	})

	t.Run("does not isolate production databases", func(t *testing.T) {
		assert.False(t, shouldIsolateTestDatabase("superplane_dev"))
		assert.False(t, shouldIsolateTestDatabase("superplane"))
	})

	t.Run("does not isolate an existing process clone", func(t *testing.T) {
		assert.False(t, shouldIsolateTestDatabase("superplane_99_test"))
	})

	t.Run("can be disabled", func(t *testing.T) {
		t.Setenv(testProcessIsolateEnv, "0")
		assert.False(t, shouldIsolateTestDatabase("superplane_test"))
	})
}

func TestIsRetryableCloneError(t *testing.T) {
	assert.True(t, isRetryableCloneError(&pgconn.PgError{Code: "55006"}))
	assert.True(t, isRetryableCloneError(fmt.Errorf("wrap: %w", &pgconn.PgError{Code: "55006"})))
	assert.False(t, isRetryableCloneError(&pgconn.PgError{Code: "42P04"}))
	assert.False(t, isRetryableCloneError(assert.AnError))
}

func TestIsolateTestDatabaseName_ClonesSharedTestDatabase(t *testing.T) {
	if os.Getenv("DB_HOST") == "" {
		t.Skip("DB_HOST not set (run with make test in Docker)")
	}

	c := dsnConfigFromEnv()
	name, err := isolateTestDatabaseName(c)
	require.NoError(t, err)
	require.True(t, isProcessTestDatabaseName(name), "got %q", name)
	require.NotEqual(t, "superplane_test", name)

	again, err := isolateTestDatabaseName(c)
	require.NoError(t, err)
	require.Equal(t, name, again, "a process must reuse one clone")
}
