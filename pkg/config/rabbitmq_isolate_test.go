package config

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcessTestVhost(t *testing.T) {
	assert.Equal(t, "superplane_12345_test", processTestVhost(12345))
}

func TestRewriteRabbitMQURLForProcess(t *testing.T) {
	t.Run("rewrites the default vhost during tests", func(t *testing.T) {
		t.Setenv("DB_NAME", "superplane_test")
		t.Setenv(testProcessIsolateEnv, "")

		got, vhost, isolate, err := rewriteRabbitMQURLForProcess("amqp://guest:guest@rabbitmq:5672", 99)
		require.NoError(t, err)
		assert.True(t, isolate)
		assert.Equal(t, "superplane_99_test", vhost)
		assert.Equal(t, "amqp://guest:guest@rabbitmq:5672/superplane_99_test", got)
	})

	t.Run("rewrites an explicit slash vhost during tests", func(t *testing.T) {
		t.Setenv("DB_NAME", "superplane_test")

		got, vhost, isolate, err := rewriteRabbitMQURLForProcess("amqp://guest:guest@rabbitmq:5672/", 7)
		require.NoError(t, err)
		assert.True(t, isolate)
		assert.Equal(t, "superplane_7_test", vhost)
		assert.Equal(t, "amqp://guest:guest@rabbitmq:5672/superplane_7_test", got)
	})

	t.Run("leaves a named vhost unchanged", func(t *testing.T) {
		t.Setenv("DB_NAME", "superplane_test")

		raw := "amqp://guest:guest@rabbitmq:5672/superplane_test"
		got, vhost, isolate, err := rewriteRabbitMQURLForProcess(raw, 99)
		require.NoError(t, err)
		assert.False(t, isolate)
		assert.Empty(t, vhost)
		assert.Equal(t, raw, got)
	})

	t.Run("leaves production URLs unchanged", func(t *testing.T) {
		t.Setenv("DB_NAME", "superplane")

		raw := "amqp://guest:guest@rabbitmq:5672"
		got, vhost, isolate, err := rewriteRabbitMQURLForProcess(raw, 99)
		require.NoError(t, err)
		assert.False(t, isolate)
		assert.Empty(t, vhost)
		assert.Equal(t, raw, got)
	})

	t.Run("can be disabled", func(t *testing.T) {
		t.Setenv("DB_NAME", "superplane_test")
		t.Setenv(testProcessIsolateEnv, "0")

		raw := "amqp://guest:guest@rabbitmq:5672"
		got, _, isolate, err := rewriteRabbitMQURLForProcess(raw, 99)
		require.NoError(t, err)
		assert.False(t, isolate)
		assert.Equal(t, raw, got)
	})
}

func TestEnsureRabbitMQVhost(t *testing.T) {
	t.Run("creates the vhost and guest permissions", func(t *testing.T) {
		var puts []string
		server := newManagementServer(t, func(method, path string) int {
			if method != "PUT" {
				return 405
			}
			puts = append(puts, path)
			return 201
		})

		require.NoError(t, ensureRabbitMQVhost(server.URL, "guest", "guest", "superplane_99_test"))
		assert.Equal(t, []string{
			"/api/vhosts/superplane_99_test",
			"/api/permissions/guest/superplane_99_test",
		}, puts)
	})

	t.Run("accepts an existing vhost", func(t *testing.T) {
		server := newManagementServer(t, func(method, path string) int {
			if path == "/api/vhosts/superplane_99_test" {
				return 204
			}
			return 201
		})

		require.NoError(t, ensureRabbitMQVhost(server.URL, "guest", "guest", "superplane_99_test"))
	})

	t.Run("returns the management error", func(t *testing.T) {
		server := newManagementServer(t, func(method, path string) int {
			return 500
		})

		err := ensureRabbitMQVhost(server.URL, "guest", "guest", "superplane_99_test")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "500")
	})
}

func TestRabbitMQURL_PassthroughNamedVhost(t *testing.T) {
	t.Setenv("DB_NAME", "superplane_test")
	t.Setenv("RABBITMQ_URL", "amqp://guest:guest@rabbitmq:5672/superplane_test")

	got, err := RabbitMQURL()
	require.NoError(t, err)
	assert.Equal(t, "amqp://guest:guest@rabbitmq:5672/superplane_test", got)
}

func TestManagementBaseURL(t *testing.T) {
	u, err := managementBaseURL("amqp://guest:secret@rabbitmq:5672/superplane_test")
	require.NoError(t, err)
	assert.Equal(t, "http://rabbitmq:15672", u)
}

func newManagementServer(t *testing.T, statusFor func(method, path string) int) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(statusFor(req.Method, req.URL.Path))
	}))
	t.Cleanup(server.Close)
	return server
}
