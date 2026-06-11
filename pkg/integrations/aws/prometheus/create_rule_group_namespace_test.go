package prometheus

import (
	"encoding/base64"
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

func Test__CreateRuleGroupNamespace__Setup(t *testing.T) {
	component := &CreateRuleGroupNamespace{}

	t.Run("missing data -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":    "us-east-1",
				"workspace": "ws-abc123",
				"name":      "application-rules",
				"data":      " ",
			},
		})

		require.ErrorContains(t, err, "rule groups YAML is required")
	})

	t.Run("valid configuration -> stores metadata", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"workspace": {
							"alias": "metrics",
							"arn": "arn:aws:aps:us-east-1:123456789012:workspace/ws-abc123",
							"status": {"statusCode": "ACTIVE"},
							"workspaceId": "ws-abc123"
						}
					}`)),
				},
			},
		}

		metadata := &contexts.MetadataContext{}
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":    "us-east-1",
				"workspace": "ws-abc123",
				"name":      "application-rules",
				"data":      "groups: []",
			},
			HTTP:        httpContext,
			Integration: validIntegrationContext(),
			Metadata:    metadata,
		})

		require.NoError(t, err)
		stored, ok := metadata.Get().(RuleGroupsNamespaceNodeMetadata)
		require.True(t, ok)
		assert.Equal(t, "us-east-1", stored.Region)
		assert.Equal(t, "ws-abc123", stored.WorkspaceID)
		assert.Equal(t, "metrics", stored.WorkspaceAlias)
		assert.Equal(t, "application-rules", stored.Namespace)
	})
}

func Test__CreateRuleGroupNamespace__Execute(t *testing.T) {
	component := &CreateRuleGroupNamespace{}

	t.Run("valid request -> emits namespace", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusAccepted,
					Body: io.NopCloser(strings.NewReader(`{
						"arn": "arn:aws:aps:us-east-1:123456789012:rulegroupsnamespace/ws-abc123/application-rules",
						"createdAt": null,
						"modifiedAt": null,
						"name": "application-rules",
						"status": {"statusCode": "CREATING"},
						"tags": {"env": "prod"}
					}`)),
				},
			},
		}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":      "us-east-1",
				"workspace":   "ws-abc123",
				"name":        " application-rules ",
				"data":        "groups: []",
				"clientToken": "token-1",
				"tags": []any{
					map[string]any{"key": "env", "value": "prod"},
				},
			},
			HTTP:           httpContext,
			ExecutionState: execState,
			Integration:    validIntegrationContext(),
		})

		require.NoError(t, err)
		assert.Equal(t, "aws.prometheus.ruleGroupNamespace", execState.Type)

		payload := execState.Payloads[0].(map[string]any)["data"].(map[string]any)
		namespace, ok := payload["ruleGroupNamespace"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "application-rules", namespace["name"])
		assert.Equal(t, "arn:aws:aps:us-east-1:123456789012:rulegroupsnamespace/ws-abc123/application-rules", namespace["arn"])
		assert.Equal(t, map[string]string{"env": "prod"}, namespace["tags"])
		assert.NotContains(t, namespace, "status")
		assert.NotContains(t, namespace, "createdAt")
		assert.NotContains(t, namespace, "modifiedAt")
		assert.Equal(t, "ws-abc123", payload["workspaceId"])

		require.Len(t, httpContext.Requests, 1)
		request := httpContext.Requests[0]
		assert.Equal(t, http.MethodPost, request.Method)
		assert.Equal(t, "https://aps.us-east-1.amazonaws.com/workspaces/ws-abc123/rulegroupsnamespaces", request.URL.String())

		requestBody, err := io.ReadAll(request.Body)
		require.NoError(t, err)

		sentPayload := map[string]any{}
		err = json.Unmarshal(requestBody, &sentPayload)
		require.NoError(t, err)
		assert.Equal(t, "application-rules", sentPayload["name"])
		assert.Equal(t, base64.StdEncoding.EncodeToString([]byte("groups: []")), sentPayload["data"])
		assert.Equal(t, "token-1", sentPayload["clientToken"])
		assert.Equal(t, map[string]any{"env": "prod"}, sentPayload["tags"])
	})
}
