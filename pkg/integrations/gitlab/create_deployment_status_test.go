package gitlab

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__CreateDeploymentStatus__Setup(t *testing.T) {
	c := &CreateDeploymentStatus{}

	t.Run("missing project", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"deploymentId": "42",
				"status":       "success",
			},
			Metadata: &contexts.MetadataContext{},
		}
		err := c.Setup(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "project is required")
	})

	t.Run("missing deployment ID", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"project": "123",
				"status":  "success",
			},
			Metadata: &contexts.MetadataContext{},
		}
		err := c.Setup(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "deployment ID is required")
	})

	t.Run("invalid status", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"project":      "123",
				"deploymentId": "42",
				"status":       "bogus",
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
				"project":      "123",
				"deploymentId": "42",
				"status":       "success",
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

func Test__CreateDeploymentStatus__Execute(t *testing.T) {
	c := &CreateDeploymentStatus{}

	t.Run("success", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{}
		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"project":      "123",
				"deploymentId": "42",
				"status":       "success",
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
					GitlabMockResponse(http.StatusOK, `{
						"id": 42,
						"iid": 2,
						"ref": "main",
						"sha": "a91957a858320c0e17f3a0eca7cfacbff50ea29a",
						"status": "success",
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
		assert.Equal(t, "success", deployment.Status)
	})

	t.Run("invalid deployment ID", func(t *testing.T) {
		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"project":      "123",
				"deploymentId": "not-a-number",
				"status":       "success",
			},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"authType":    AuthTypePersonalAccessToken,
					"groupId":     "123",
					"accessToken": "pat",
					"baseUrl":     "https://gitlab.com",
				},
			},
		}

		err := c.Execute(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid deployment ID")
	})

	t.Run("failure", func(t *testing.T) {
		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"project":      "123",
				"deploymentId": "99",
				"status":       "failed",
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
					GitlabMockResponse(http.StatusNotFound, `{"message": "404 Deployment Not Found"}`),
				},
			},
		}

		err := c.Execute(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to update deployment status")
	})
}
