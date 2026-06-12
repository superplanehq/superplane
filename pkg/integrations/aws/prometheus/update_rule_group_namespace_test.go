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

func Test__UpdateRuleGroupNamespace__Setup(t *testing.T) {
	component := &UpdateRuleGroupNamespace{}

	t.Run("missing data -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":    "us-east-1",
				"workspace": "ws-abc123",
				"namespace": "application-rules",
				"data":      " ",
			},
		})

		require.ErrorContains(t, err, "rule groups YAML is required")
	})

	t.Run("valid configuration -> ok", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":    "us-east-1",
				"workspace": "ws-abc123",
				"namespace": "application-rules",
				"data":      "groups: []",
			},
		})

		require.NoError(t, err)
	})
}

func Test__UpdateRuleGroupNamespace__Execute(t *testing.T) {
	component := &UpdateRuleGroupNamespace{}

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
						"status": {"statusCode": "UPDATING"}
					}`)),
				},
			},
		}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":      "us-east-1",
				"workspace":   "ws-abc123",
				"namespace":   " application-rules ",
				"data":        "groups: []",
				"clientToken": "token-1",
			},
			HTTP:           httpContext,
			ExecutionState: execState,
			Integration:    validIntegrationContext(),
		})

		require.NoError(t, err)
		assert.Equal(t, "aws.prometheus.ruleGroupNamespace.updated", execState.Type)

		payload := execState.Payloads[0].(map[string]any)["data"].(map[string]any)
		namespace, ok := payload["ruleGroupNamespace"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "application-rules", namespace["name"])
		assert.Equal(t, "arn:aws:aps:us-east-1:123456789012:rulegroupsnamespace/ws-abc123/application-rules", namespace["arn"])
		assert.NotContains(t, namespace, "status")
		assert.NotContains(t, namespace, "createdAt")
		assert.NotContains(t, namespace, "modifiedAt")

		require.Len(t, httpContext.Requests, 1)
		request := httpContext.Requests[0]
		assert.Equal(t, http.MethodPut, request.Method)
		assert.Equal(t, "https://aps.us-east-1.amazonaws.com/workspaces/ws-abc123/rulegroupsnamespaces/application-rules", request.URL.String())

		requestBody, err := io.ReadAll(request.Body)
		require.NoError(t, err)

		sentPayload := map[string]any{}
		err = json.Unmarshal(requestBody, &sentPayload)
		require.NoError(t, err)
		assert.Equal(t, base64.StdEncoding.EncodeToString([]byte("groups: []")), sentPayload["data"])
		assert.Equal(t, "token-1", sentPayload["clientToken"])
	})
}
