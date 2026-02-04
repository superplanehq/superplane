package dockerhub

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__DockerHub__Sync(t *testing.T) {
	integration := &DockerHub{}

	t.Run("missing username -> error", func(t *testing.T) {
		err := integration.Sync(core.SyncContext{
			Configuration: map[string]any{
				"accessToken": "test-token",
			},
		})

		require.ErrorContains(t, err, "username is required")
	})

	t.Run("missing accessToken -> error", func(t *testing.T) {
		err := integration.Sync(core.SyncContext{
			Configuration: map[string]any{
				"username": "testuser",
			},
		})

		require.ErrorContains(t, err, "accessToken is required")
	})

	t.Run("invalid credentials -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusUnauthorized,
					Body:       io.NopCloser(strings.NewReader(`{"message": "invalid credentials"}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"username":    "testuser",
				"accessToken": "invalid-token",
			},
		}

		err := integration.Sync(core.SyncContext{
			Configuration: map[string]any{
				"username":    "testuser",
				"accessToken": "invalid-token",
			},
			HTTP:        httpContext,
			Integration: appCtx,
		})

		require.ErrorContains(t, err, "invalid credentials")
	})

	t.Run("valid credentials -> success", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"token": "jwt-token"}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"username":    "testuser",
				"accessToken": "valid-token",
			},
		}

		err := integration.Sync(core.SyncContext{
			Configuration: map[string]any{
				"username":    "testuser",
				"accessToken": "valid-token",
			},
			HTTP:        httpContext,
			Integration: appCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", appCtx.State)

		// Verify metadata was set
		m := appCtx.Metadata.(Metadata)
		assert.Equal(t, "testuser", m.Username)
	})
}

func Test__DockerHub__CompareWebhookConfig(t *testing.T) {
	integration := &DockerHub{}

	t.Run("same repository -> equal", func(t *testing.T) {
		a := WebhookConfiguration{Repository: "myorg/myapp"}
		b := WebhookConfiguration{Repository: "myorg/myapp"}

		equal, err := integration.CompareWebhookConfig(a, b)

		require.NoError(t, err)
		assert.True(t, equal)
	})

	t.Run("different repository -> not equal", func(t *testing.T) {
		a := WebhookConfiguration{Repository: "myorg/myapp"}
		b := WebhookConfiguration{Repository: "other/repo"}

		equal, err := integration.CompareWebhookConfig(a, b)

		require.NoError(t, err)
		assert.False(t, equal)
	})
}

func Test__DockerHub__InterfaceMethods(t *testing.T) {
	integration := &DockerHub{}

	t.Run("Name returns dockerhub", func(t *testing.T) {
		assert.Equal(t, "dockerhub", integration.Name())
	})

	t.Run("Label returns Docker Hub", func(t *testing.T) {
		assert.Equal(t, "Docker Hub", integration.Label())
	})

	t.Run("Icon returns docker", func(t *testing.T) {
		assert.Equal(t, "docker", integration.Icon())
	})

	t.Run("Components returns ListTags", func(t *testing.T) {
		components := integration.Components()
		require.Len(t, components, 1)
		assert.Equal(t, "dockerhub.listTags", components[0].Name())
	})

	t.Run("Triggers returns OnImagePushed", func(t *testing.T) {
		triggers := integration.Triggers()
		require.Len(t, triggers, 1)
		assert.Equal(t, "dockerhub.onImagePushed", triggers[0].Name())
	})
}
