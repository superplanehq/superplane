package gitlab

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__CreateDeployment__Setup(t *testing.T) {
	c := &CreateDeployment{}

	t.Run("missing project", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"environment": "production",
				"ref":         "main",
				"sha":         "abc123",
			},
			Metadata: &contexts.MetadataContext{},
		}
		err := c.Setup(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "project is required")
	})

	t.Run("missing environment", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"project": "123",
				"ref":     "main",
				"sha":     "abc123",
			},
			Metadata: &contexts.MetadataContext{},
		}
		err := c.Setup(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "environment is required")
	})

	t.Run("missing ref", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"project":     "123",
				"environment": "production",
				"sha":         "abc123",
			},
			Metadata: &contexts.MetadataContext{},
		}
		err := c.Setup(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ref is required")
	})

	t.Run("missing sha", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"project":     "123",
				"environment": "production",
				"ref":         "main",
			},
			Metadata: &contexts.MetadataContext{},
		}
		err := c.Setup(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "commit SHA is required")
	})

	t.Run("invalid status", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"project":     "123",
				"environment": "production",
				"ref":         "main",
				"sha":         "abc123",
				"status":      "bogus",
			},
			Metadata: &contexts.MetadataContext{},
		}
		err := c.Setup(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid status")
	})

	t.Run("valid configuration", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"project":     "123",
				"environment": "production",
				"ref":         "main",
				"sha":         "abc123",
				"status":      "running",
			},
			Integration: &contexts.IntegrationContext{
				Metadata: Metadata{
					Projects: []ProjectMetadata{
						{ID: 123, Name: "repo", URL: "http://repo"},
					},
				},
			},
			Metadata: &contexts.MetadataContext{},
		}
		err := c.Setup(ctx)
		require.NoError(t, err)
	})
}

func Test__CreateDeployment__Execute(t *testing.T) {
	c := &CreateDeployment{}

	t.Run("success", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{}
		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"project":     "123",
				"environment": "production",
				"ref":         "main",
				"sha":         "a91957a858320c0e17f3a0eca7cfacbff50ea29a",
				"status":      "running",
			},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"authType":    AuthTypePersonalAccessToken,
					"groupId":     "123",
					"accessToken": "pat",
					"baseUrl":     "https://gitlab.com",
				},
			},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					GitlabMockResponse(http.StatusCreated, `{
						"id": 42,
						"iid": 2,
						"ref": "main",
						"sha": "a91957a858320c0e17f3a0eca7cfacbff50ea29a",
						"status": "running",
						"environment": {"id": 9, "name": "production", "external_url": "https://prod.example.com"}
					}`),
				},
			},
			ExecutionState: executionState,
		}

		err := c.Execute(ctx)
		require.NoError(t, err)

		require.Len(t, executionState.Payloads, 1)
		payload := executionState.Payloads[0].(map[string]any)
		assert.Equal(t, core.DefaultOutputChannel.Name, executionState.Channel)
		assert.Equal(t, DeploymentPayloadType, executionState.Type)

		var deployment Deployment
		deploymentPayload := payload["data"]
		payloadBytes, _ := json.Marshal(deploymentPayload)
		require.NoError(t, json.Unmarshal(payloadBytes, &deployment))

		assert.Equal(t, 42, deployment.ID)
		assert.Equal(t, "running", deployment.Status)
		require.NotNil(t, deployment.Environment)
		assert.Equal(t, "production", deployment.Environment.Name)
	})

	t.Run("infers tag from git-ref tag prefix even when tag checkbox is off", func(t *testing.T) {
		mockHTTP := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusCreated, `{
					"id": 42,
					"ref": "v1.0.0",
					"status": "running",
					"environment": {"id": 9, "name": "production"}
				}`),
			},
		}

		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"project":     "123",
				"environment": "production",
				"ref":         "refs/tags/v1.0.0",
				"sha":         "a91957a858320c0e17f3a0eca7cfacbff50ea29a",
				"status":      "running",
				"tag":         false,
			},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"authType":    AuthTypePersonalAccessToken,
					"groupId":     "123",
					"accessToken": "pat",
					"baseUrl":     "https://gitlab.com",
				},
			},
			HTTP:           mockHTTP,
			ExecutionState: &contexts.ExecutionStateContext{},
		}

		err := c.Execute(ctx)
		require.NoError(t, err)

		require.Len(t, mockHTTP.Requests, 1)
		body, readErr := io.ReadAll(mockHTTP.Requests[0].Body)
		require.NoError(t, readErr)
		bodyString := string(body)
		assert.Contains(t, bodyString, `"ref":"v1.0.0"`)
		assert.Contains(t, bodyString, `"tag":true`)
	})

	t.Run("failure", func(t *testing.T) {
		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"project":     "123",
				"environment": "production",
				"ref":         "main",
				"sha":         "abc123",
				"status":      "running",
			},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"authType":    AuthTypePersonalAccessToken,
					"groupId":     "123",
					"accessToken": "pat",
					"baseUrl":     "https://gitlab.com",
				},
			},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					GitlabMockResponse(http.StatusBadRequest, `{"message": "sha is invalid"}`),
				},
			},
		}

		err := c.Execute(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create deployment")
	})
}
