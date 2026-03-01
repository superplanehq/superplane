package octopus

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__Octopus_DeployRelease__Setup(t *testing.T) {
	component := &DeployRelease{}

	t.Run("missing project -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"release":     "Releases-1",
				"environment": "Environments-1",
			},
		})

		require.ErrorContains(t, err, "project is required")
	})

	t.Run("missing release -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"project":     "Projects-1",
				"environment": "Environments-1",
			},
		})

		require.ErrorContains(t, err, "release is required")
	})

	t.Run("missing environment -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"project": "Projects-1",
				"release": "Releases-1",
			},
		})

		require.ErrorContains(t, err, "environment is required")
	})

	t.Run("valid configuration -> requests webhook and stores metadata", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"serverUrl": "https://octopus.example.com",
				"apiKey":    "API-TEST",
			},
		}
		metadataCtx := &contexts.MetadataContext{}

		err := component.Setup(core.SetupContext{
			Integration: integrationCtx,
			Metadata:    metadataCtx,
			Configuration: map[string]any{
				"project":     "Projects-1",
				"release":     "Releases-10",
				"environment": "Environments-2",
			},
		})

		require.NoError(t, err)
		require.Len(t, integrationCtx.WebhookRequests, 1)

		webhookConfig, ok := integrationCtx.WebhookRequests[0].(WebhookConfiguration)
		require.True(t, ok)
		assert.Equal(t, []string{
			EventCategoryDeploymentSucceeded,
			EventCategoryDeploymentFailed,
		}, webhookConfig.EventCategories)
	})
}

func Test__Octopus_DeployRelease__Execute(t *testing.T) {
	component := &DeployRelease{}

	t.Run("creates deployment and schedules poll", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				// ListSpaces (for spaceIDForIntegration)
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`[{"Id":"Spaces-1","Name":"Default","IsDefault":true}]`,
					)),
				},
				// CreateDeployment
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`{"Id":"Deployments-100","TaskId":"ServerTasks-200","ProjectId":"Projects-1","ReleaseId":"Releases-10","EnvironmentId":"Environments-2","Created":"2026-01-15T10:00:00Z"}`,
					)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"serverUrl": "https://octopus.example.com",
				"apiKey":    "API-TEST",
			},
		}
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		metadataCtx := &contexts.MetadataContext{}
		requestCtx := &contexts.RequestContext{}

		err := component.Execute(core.ExecutionContext{
			HTTP:           httpCtx,
			Integration:    integrationCtx,
			Metadata:       metadataCtx,
			ExecutionState: executionState,
			Requests:       requestCtx,
			Configuration: map[string]any{
				"project":     "Projects-1",
				"release":     "Releases-10",
				"environment": "Environments-2",
			},
		})

		require.NoError(t, err)

		// Verify deployment ID stored in KV
		assert.Equal(t, "Deployments-100", executionState.KVs["deployment_id"])

		// Verify poll scheduled
		assert.Equal(t, "poll", requestCtx.Action)
		assert.Equal(t, DeployReleasePollInterval, requestCtx.Duration)

		// Verify execution state not yet finished
		assert.Empty(t, executionState.Channel)

		// Verify metadata was stored
		require.NotNil(t, metadataCtx.Metadata)

		// Verify HTTP requests: ListSpaces + CreateDeployment
		require.Len(t, httpCtx.Requests, 2)
		assert.Equal(t, http.MethodGet, httpCtx.Requests[0].Method)
		assert.Contains(t, httpCtx.Requests[0].URL.Path, "/api/spaces/all")

		assert.Equal(t, http.MethodPost, httpCtx.Requests[1].Method)
		assert.Contains(t, httpCtx.Requests[1].URL.Path, "/api/Spaces-1/deployments")

		// Verify deployment request body
		reqBody, readErr := io.ReadAll(httpCtx.Requests[1].Body)
		require.NoError(t, readErr)
		reqPayload := map[string]any{}
		require.NoError(t, json.Unmarshal(reqBody, &reqPayload))
		assert.Equal(t, "Releases-10", reqPayload["ReleaseId"])
		assert.Equal(t, "Environments-2", reqPayload["EnvironmentId"])
	})

	t.Run("missing required fields -> error", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"serverUrl": "https://octopus.example.com",
					"apiKey":    "API-TEST",
				},
			},
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
			Configuration:  map[string]any{},
		})

		require.ErrorContains(t, err, "project is required")
	})

	t.Run("API error -> returns error", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				// ListSpaces
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`[{"Id":"Spaces-1","Name":"Default","IsDefault":true}]`,
					)),
				},
				// CreateDeployment fails
				{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(strings.NewReader(`{"ErrorMessage":"Release not found"}`)),
				},
			},
		}

		err := component.Execute(core.ExecutionContext{
			HTTP: httpCtx,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"serverUrl": "https://octopus.example.com",
					"apiKey":    "API-TEST",
				},
			},
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
			Metadata:       &contexts.MetadataContext{},
			Requests:       &contexts.RequestContext{},
			Configuration: map[string]any{
				"project":     "Projects-1",
				"release":     "Releases-999",
				"environment": "Environments-2",
			},
		})

		require.Error(t, err)
	})
}

func Test__Octopus_DeployRelease__HandleWebhook(t *testing.T) {
	component := &DeployRelease{}

	secret := "test-webhook-secret"

	t.Run("deployment succeeded -> emits to success channel", func(t *testing.T) {
		payload := map[string]any{
			"Timestamp": "2026-01-15T10:35:00.000Z",
			"EventType": "SubscriptionPayload",
			"Payload": map[string]any{
				"Event": map[string]any{
					"Category": "DeploymentSucceeded",
					"RelatedDocumentIds": []any{
						"Deployments-100",
						"Projects-1",
					},
				},
			},
		}
		body, marshalErr := json.Marshal(payload)
		require.NoError(t, marshalErr)

		headers := http.Header{
			"Content-Type":                []string{"application/json"},
			"X-Superplane-Webhook-Secret": []string{secret},
		}

		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		metadataCtx := &contexts.MetadataContext{
			Metadata: map[string]any{
				"deployment": map[string]any{
					"id":            "Deployments-100",
					"taskId":        "ServerTasks-200",
					"taskState":     "Executing",
					"projectId":     "Projects-1",
					"releaseId":     "Releases-10",
					"environmentId": "Environments-2",
					"created":       "2026-01-15T10:00:00Z",
				},
			},
		}

		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				// ListSpaces (for NewClient -> spaceIDForIntegration)
				// GetTask
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`{"Id":"ServerTasks-200","State":"Success","IsCompleted":true,"FinishedSuccessfully":true,"CompletedTime":"2026-01-15T10:35:00Z","Duration":"5m"}`,
					)),
				},
			},
		}

		status, webhookErr := component.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			HTTP:    httpCtx,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"serverUrl": "https://octopus.example.com",
					"apiKey":    "API-TEST",
				},
			},
			Webhook: &contexts.NodeWebhookContext{Secret: secret},
			FindExecutionByKV: func(key, value string) (*core.ExecutionContext, error) {
				if key == "deployment_id" && value == "Deployments-100" {
					return &core.ExecutionContext{
						Metadata:       metadataCtx,
						ExecutionState: executionState,
					}, nil
				}
				return nil, assert.AnError
			},
		})

		assert.Equal(t, http.StatusOK, status)
		require.NoError(t, webhookErr)
		assert.Equal(t, DeployReleaseSuccessOutputChannel, executionState.Channel)
		assert.Equal(t, DeployReleasePayloadType, executionState.Type)
		require.Len(t, executionState.Payloads, 1)

		emittedPayload := readMap(executionState.Payloads[0])
		data := readMap(emittedPayload["data"])
		assert.Equal(t, "Deployments-100", data["deploymentId"])
		assert.Equal(t, "Success", data["taskState"])
	})

	t.Run("deployment failed -> emits to failed channel", func(t *testing.T) {
		payload := map[string]any{
			"Timestamp": "2026-01-15T10:35:00.000Z",
			"EventType": "SubscriptionPayload",
			"Payload": map[string]any{
				"Event": map[string]any{
					"Category": "DeploymentFailed",
					"RelatedDocumentIds": []any{
						"Deployments-200",
						"Projects-1",
					},
				},
			},
		}
		body, marshalErr := json.Marshal(payload)
		require.NoError(t, marshalErr)

		headers := http.Header{
			"Content-Type":                []string{"application/json"},
			"X-Superplane-Webhook-Secret": []string{secret},
		}

		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		metadataCtx := &contexts.MetadataContext{
			Metadata: map[string]any{
				"deployment": map[string]any{
					"id":            "Deployments-200",
					"taskId":        "ServerTasks-300",
					"taskState":     "Executing",
					"projectId":     "Projects-1",
					"releaseId":     "Releases-10",
					"environmentId": "Environments-2",
					"created":       "2026-01-15T10:00:00Z",
				},
			},
		}

		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				// GetTask
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`{"Id":"ServerTasks-300","State":"Failed","IsCompleted":true,"FinishedSuccessfully":false,"CompletedTime":"2026-01-15T10:35:00Z","ErrorMessage":"Deploy script failed","Duration":"5m"}`,
					)),
				},
			},
		}

		status, webhookErr := component.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			HTTP:    httpCtx,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"serverUrl": "https://octopus.example.com",
					"apiKey":    "API-TEST",
				},
			},
			Webhook: &contexts.NodeWebhookContext{Secret: secret},
			FindExecutionByKV: func(key, value string) (*core.ExecutionContext, error) {
				if key == "deployment_id" && value == "Deployments-200" {
					return &core.ExecutionContext{
						Metadata:       metadataCtx,
						ExecutionState: executionState,
					}, nil
				}
				return nil, assert.AnError
			},
		})

		assert.Equal(t, http.StatusOK, status)
		require.NoError(t, webhookErr)
		assert.Equal(t, DeployReleaseFailedOutputChannel, executionState.Channel)
		assert.Equal(t, DeployReleasePayloadType, executionState.Type)

		emittedPayload := readMap(executionState.Payloads[0])
		data := readMap(emittedPayload["data"])
		assert.Equal(t, "Deployments-200", data["deploymentId"])
		assert.Equal(t, "Failed", data["taskState"])
	})

	t.Run("non-deployment event type -> 200, no action", func(t *testing.T) {
		payload := map[string]any{
			"Timestamp": "2026-01-15T10:35:00.000Z",
			"EventType": "SubscriptionPayload",
			"Payload": map[string]any{
				"Event": map[string]any{
					"Category": "DeploymentStarted",
				},
			},
		}
		body, marshalErr := json.Marshal(payload)
		require.NoError(t, marshalErr)

		headers := http.Header{
			"Content-Type":                []string{"application/json"},
			"X-Superplane-Webhook-Secret": []string{secret},
		}

		status, webhookErr := component.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Webhook: &contexts.NodeWebhookContext{Secret: secret},
		})

		assert.Equal(t, http.StatusOK, status)
		require.NoError(t, webhookErr)
	})

	t.Run("no deployment IDs in event -> 200, no action", func(t *testing.T) {
		payload := map[string]any{
			"Timestamp": "2026-01-15T10:35:00.000Z",
			"EventType": "SubscriptionPayload",
			"Payload": map[string]any{
				"Event": map[string]any{
					"Category":           "DeploymentSucceeded",
					"RelatedDocumentIds": []any{},
				},
			},
		}
		body, marshalErr := json.Marshal(payload)
		require.NoError(t, marshalErr)

		headers := http.Header{
			"Content-Type":                []string{"application/json"},
			"X-Superplane-Webhook-Secret": []string{secret},
		}

		status, webhookErr := component.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Webhook: &contexts.NodeWebhookContext{Secret: secret},
		})

		assert.Equal(t, http.StatusOK, status)
		require.NoError(t, webhookErr)
	})

	t.Run("deployment not found by KV -> 200, no action", func(t *testing.T) {
		payload := map[string]any{
			"Timestamp": "2026-01-15T10:35:00.000Z",
			"EventType": "SubscriptionPayload",
			"Payload": map[string]any{
				"Event": map[string]any{
					"Category":           "DeploymentSucceeded",
					"RelatedDocumentIds": []any{"Deployments-999"},
				},
			},
		}
		body, marshalErr := json.Marshal(payload)
		require.NoError(t, marshalErr)

		headers := http.Header{
			"Content-Type":                []string{"application/json"},
			"X-Superplane-Webhook-Secret": []string{secret},
		}

		status, webhookErr := component.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Webhook: &contexts.NodeWebhookContext{Secret: secret},
			FindExecutionByKV: func(key, value string) (*core.ExecutionContext, error) {
				return nil, assert.AnError
			},
		})

		assert.Equal(t, http.StatusOK, status)
		require.NoError(t, webhookErr)
	})

	t.Run("first deployment ID errors, second matches -> emits result", func(t *testing.T) {
		payload := map[string]any{
			"Timestamp": "2026-01-15T10:35:00.000Z",
			"EventType": "SubscriptionPayload",
			"Payload": map[string]any{
				"Event": map[string]any{
					"Category": "DeploymentSucceeded",
					"RelatedDocumentIds": []any{
						"Deployments-999",
						"Deployments-100",
						"Projects-1",
					},
				},
			},
		}
		body, marshalErr := json.Marshal(payload)
		require.NoError(t, marshalErr)

		headers := http.Header{
			"Content-Type":                []string{"application/json"},
			"X-Superplane-Webhook-Secret": []string{secret},
		}

		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		metadataCtx := &contexts.MetadataContext{
			Metadata: map[string]any{
				"deployment": map[string]any{
					"id":            "Deployments-100",
					"taskId":        "ServerTasks-200",
					"taskState":     "Executing",
					"projectId":     "Projects-1",
					"releaseId":     "Releases-10",
					"environmentId": "Environments-2",
					"created":       "2026-01-15T10:00:00Z",
				},
			},
		}

		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				// GetTask
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`{"Id":"ServerTasks-200","State":"Success","IsCompleted":true,"FinishedSuccessfully":true,"CompletedTime":"2026-01-15T10:35:00Z","Duration":"5m"}`,
					)),
				},
			},
		}

		status, webhookErr := component.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			HTTP:    httpCtx,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"serverUrl": "https://octopus.example.com",
					"apiKey":    "API-TEST",
				},
			},
			Webhook: &contexts.NodeWebhookContext{Secret: secret},
			FindExecutionByKV: func(key, value string) (*core.ExecutionContext, error) {
				if key == "deployment_id" && value == "Deployments-100" {
					return &core.ExecutionContext{
						Metadata:       metadataCtx,
						ExecutionState: executionState,
					}, nil
				}
				return nil, assert.AnError
			},
		})

		assert.Equal(t, http.StatusOK, status)
		require.NoError(t, webhookErr)
		assert.Equal(t, DeployReleaseSuccessOutputChannel, executionState.Channel)
	})

	t.Run("already completed deployment -> 200, no action", func(t *testing.T) {
		payload := map[string]any{
			"Timestamp": "2026-01-15T10:35:00.000Z",
			"EventType": "SubscriptionPayload",
			"Payload": map[string]any{
				"Event": map[string]any{
					"Category":           "DeploymentSucceeded",
					"RelatedDocumentIds": []any{"Deployments-100"},
				},
			},
		}
		body, marshalErr := json.Marshal(payload)
		require.NoError(t, marshalErr)

		headers := http.Header{
			"Content-Type":                []string{"application/json"},
			"X-Superplane-Webhook-Secret": []string{secret},
		}

		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		metadataCtx := &contexts.MetadataContext{
			Metadata: map[string]any{
				"deployment": map[string]any{
					"id":            "Deployments-100",
					"taskId":        "ServerTasks-200",
					"taskState":     "Success",
					"completedTime": "2026-01-15T10:30:00Z",
				},
			},
		}

		status, webhookErr := component.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Webhook: &contexts.NodeWebhookContext{Secret: secret},
			FindExecutionByKV: func(key, value string) (*core.ExecutionContext, error) {
				if key == "deployment_id" && value == "Deployments-100" {
					return &core.ExecutionContext{
						Metadata:       metadataCtx,
						ExecutionState: executionState,
					}, nil
				}
				return nil, assert.AnError
			},
		})

		assert.Equal(t, http.StatusOK, status)
		require.NoError(t, webhookErr)
		assert.Empty(t, executionState.Channel)
	})
}

func Test__Octopus_DeployRelease__Poll(t *testing.T) {
	component := &DeployRelease{}

	t.Run("task still executing -> reschedules poll", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				// GetTask
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`{"Id":"ServerTasks-200","State":"Executing","IsCompleted":false}`,
					)),
				},
			},
		}

		requestCtx := &contexts.RequestContext{}
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		metadataCtx := &contexts.MetadataContext{
			Metadata: map[string]any{
				"deployment": map[string]any{
					"id":     "Deployments-100",
					"taskId": "ServerTasks-200",
				},
			},
		}

		err := component.HandleAction(core.ActionContext{
			Name: "poll",
			HTTP: httpCtx,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"serverUrl": "https://octopus.example.com",
					"apiKey":    "API-TEST",
				},
			},
			ExecutionState: executionState,
			Metadata:       metadataCtx,
			Requests:       requestCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, "poll", requestCtx.Action)
		assert.Equal(t, DeployReleasePollInterval, requestCtx.Duration)
		assert.Empty(t, executionState.Channel)
	})

	t.Run("task succeeded -> emits to success channel", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				// GetTask
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`{"Id":"ServerTasks-200","State":"Success","IsCompleted":true,"FinishedSuccessfully":true,"CompletedTime":"2026-01-15T10:35:00Z","Duration":"5m"}`,
					)),
				},
			},
		}

		requestCtx := &contexts.RequestContext{}
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		metadataCtx := &contexts.MetadataContext{
			Metadata: map[string]any{
				"deployment": map[string]any{
					"id":            "Deployments-100",
					"taskId":        "ServerTasks-200",
					"projectId":     "Projects-1",
					"releaseId":     "Releases-10",
					"environmentId": "Environments-2",
					"created":       "2026-01-15T10:00:00Z",
				},
			},
		}

		err := component.HandleAction(core.ActionContext{
			Name: "poll",
			HTTP: httpCtx,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"serverUrl": "https://octopus.example.com",
					"apiKey":    "API-TEST",
				},
			},
			ExecutionState: executionState,
			Metadata:       metadataCtx,
			Requests:       requestCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, DeployReleaseSuccessOutputChannel, executionState.Channel)
		assert.Equal(t, DeployReleasePayloadType, executionState.Type)
		require.Len(t, executionState.Payloads, 1)

		emittedPayload := readMap(executionState.Payloads[0])
		data := readMap(emittedPayload["data"])
		assert.Equal(t, "Deployments-100", data["deploymentId"])
		assert.Equal(t, "Success", data["taskState"])
		assert.Equal(t, "2026-01-15T10:35:00Z", data["completedTime"])
	})

	t.Run("task failed -> emits to failed channel", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				// GetTask
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`{"Id":"ServerTasks-200","State":"Failed","IsCompleted":true,"FinishedSuccessfully":false,"CompletedTime":"2026-01-15T10:35:00Z","ErrorMessage":"Script error","Duration":"5m"}`,
					)),
				},
			},
		}

		requestCtx := &contexts.RequestContext{}
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		metadataCtx := &contexts.MetadataContext{
			Metadata: map[string]any{
				"deployment": map[string]any{
					"id":            "Deployments-100",
					"taskId":        "ServerTasks-200",
					"projectId":     "Projects-1",
					"releaseId":     "Releases-10",
					"environmentId": "Environments-2",
					"created":       "2026-01-15T10:00:00Z",
				},
			},
		}

		err := component.HandleAction(core.ActionContext{
			Name: "poll",
			HTTP: httpCtx,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"serverUrl": "https://octopus.example.com",
					"apiKey":    "API-TEST",
				},
			},
			ExecutionState: executionState,
			Metadata:       metadataCtx,
			Requests:       requestCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, DeployReleaseFailedOutputChannel, executionState.Channel)
		assert.Equal(t, DeployReleasePayloadType, executionState.Type)

		emittedPayload := readMap(executionState.Payloads[0])
		data := readMap(emittedPayload["data"])
		assert.Equal(t, "Failed", data["taskState"])
		assert.Equal(t, "Script error", data["errorMessage"])
	})

	t.Run("already finished execution -> no-op", func(t *testing.T) {
		requestCtx := &contexts.RequestContext{}
		executionState := &contexts.ExecutionStateContext{
			Finished: true,
			KVs:      map[string]string{},
		}
		metadataCtx := &contexts.MetadataContext{
			Metadata: map[string]any{
				"deployment": map[string]any{
					"id":     "Deployments-100",
					"taskId": "ServerTasks-200",
				},
			},
		}

		err := component.HandleAction(core.ActionContext{
			Name:           "poll",
			ExecutionState: executionState,
			Metadata:       metadataCtx,
			Requests:       requestCtx,
		})

		require.NoError(t, err)
		assert.Empty(t, requestCtx.Action)
	})

	t.Run("unknown action -> error", func(t *testing.T) {
		err := component.HandleAction(core.ActionContext{
			Name: "unknown",
		})

		require.ErrorContains(t, err, "unknown action: unknown")
	})
}

func Test__Octopus_DeployRelease__Cancel(t *testing.T) {
	component := &DeployRelease{}

	t.Run("already completed -> no-op", func(t *testing.T) {
		metadataCtx := &contexts.MetadataContext{
			Metadata: map[string]any{
				"deployment": map[string]any{
					"id":        "Deployments-100",
					"taskId":    "ServerTasks-200",
					"taskState": "Success",
				},
			},
		}

		err := component.Cancel(core.ExecutionContext{
			Metadata:    metadataCtx,
			Integration: &contexts.IntegrationContext{},
		})

		require.NoError(t, err)
	})

	t.Run("no metadata -> no-op", func(t *testing.T) {
		metadataCtx := &contexts.MetadataContext{
			Metadata: map[string]any{},
		}

		err := component.Cancel(core.ExecutionContext{
			Metadata:    metadataCtx,
			Integration: &contexts.IntegrationContext{},
		})

		require.NoError(t, err)
	})

	t.Run("active deployment -> cancels task", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				// ListSpaces (for spaceIDForIntegration)
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`[{"Id":"Spaces-1","Name":"Default","IsDefault":true}]`,
					)),
				},
				// CancelTask
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{}`)),
				},
			},
		}

		metadataCtx := &contexts.MetadataContext{
			Metadata: map[string]any{
				"deployment": map[string]any{
					"id":        "Deployments-100",
					"taskId":    "ServerTasks-200",
					"taskState": "Executing",
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"serverUrl": "https://octopus.example.com",
				"apiKey":    "API-TEST",
			},
		}

		err := component.Cancel(core.ExecutionContext{
			HTTP:        httpCtx,
			Metadata:    metadataCtx,
			Integration: integrationCtx,
		})

		require.NoError(t, err)

		// Verify HTTP requests: ListSpaces + CancelTask
		require.Len(t, httpCtx.Requests, 2)
		assert.Equal(t, http.MethodGet, httpCtx.Requests[0].Method)
		assert.Contains(t, httpCtx.Requests[0].URL.Path, "/api/spaces/all")

		assert.Equal(t, http.MethodPost, httpCtx.Requests[1].Method)
		assert.Contains(t, httpCtx.Requests[1].URL.Path, "/api/Spaces-1/tasks/ServerTasks-200/cancel")
	})
}
