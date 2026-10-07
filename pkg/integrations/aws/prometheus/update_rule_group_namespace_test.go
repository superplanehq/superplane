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

	t.Run("invalid configuration -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: "invalid",
		})

		require.ErrorContains(t, err, "failed to decode configuration")
	})

	t.Run("missing region -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"workspace": "ws-abc123",
				"namespace": "example-alerts",
				"data":      validRulesYAML,
			},
		})

		require.ErrorContains(t, err, "region is required")
	})

	t.Run("missing workspace -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":    "us-east-1",
				"namespace": "example-alerts",
				"data":      validRulesYAML,
			},
		})

		require.ErrorContains(t, err, "workspace is required")
	})

	t.Run("missing namespace -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":    "us-east-1",
				"workspace": "ws-abc123",
				"data":      validRulesYAML,
			},
		})

		require.ErrorContains(t, err, "namespace is required")
	})

	t.Run("missing data -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":    "us-east-1",
				"workspace": "ws-abc123",
				"namespace": "example-alerts",
			},
		})

		require.ErrorContains(t, err, "data is required")
	})

	t.Run("malformed YAML -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":    "us-east-1",
				"workspace": "ws-abc123",
				"namespace": "example-alerts",
				"data":      "groups: [unterminated",
			},
		})

		require.ErrorContains(t, err, "data is not valid YAML")
	})

	t.Run("valid configuration -> ok", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":    "us-east-1",
				"workspace": "ws-abc123",
				"namespace": "example-alerts",
				"data":      validRulesYAML,
			},
		})

		require.NoError(t, err)
	})
}

func Test__UpdateRuleGroupNamespace__Execute(t *testing.T) {
	component := &UpdateRuleGroupNamespace{}

	t.Run("valid request -> emits namespace and base64-encodes the rules document", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusAccepted,
					Body: io.NopCloser(strings.NewReader(`{
						"arn": "arn:aws:aps:us-east-1:123456789012:rulegroupsnamespace/ws-abc123/example-alerts",
						"name": "example-alerts",
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
				"namespace":   "example-alerts",
				"data":        validRulesYAML,
				"clientToken": "token-1",
			},
			HTTP:           httpContext,
			ExecutionState: execState,
			Integration:    validIntegrationContext(),
		})

		require.NoError(t, err)
		assert.True(t, execState.Finished)
		assert.True(t, execState.Passed)
		assert.Equal(t, core.DefaultOutputChannel.Name, execState.Channel)
		assert.Equal(t, "aws.prometheus.ruleGroupNamespace.updated", execState.Type)

		require.Len(t, execState.Payloads, 1)
		payload := execState.Payloads[0].(map[string]any)["data"].(map[string]any)
		assert.Equal(t, "ws-abc123", payload["workspaceId"])
		namespace, ok := payload["namespace"].(*CreateRuleGroupsNamespaceResponse)
		require.True(t, ok)
		assert.Equal(t, "example-alerts", namespace.Name)
		assert.Equal(t, "UPDATING", namespace.Status.StatusCode)

		require.Len(t, httpContext.Requests, 1)
		request := httpContext.Requests[0]
		assert.Equal(t, http.MethodPut, request.Method)
		assert.Equal(t, "https://aps.us-east-1.amazonaws.com/workspaces/ws-abc123/rulegroupsnamespaces/example-alerts", request.URL.String())

		requestBody, err := io.ReadAll(request.Body)
		require.NoError(t, err)

		sentPayload := map[string]any{}
		err = json.Unmarshal(requestBody, &sentPayload)
		require.NoError(t, err)
		assert.Equal(t, "token-1", sentPayload["clientToken"])
		assert.Equal(t, base64.StdEncoding.EncodeToString([]byte(strings.TrimSpace(validRulesYAML))), sentPayload["data"])
		assert.NotContains(t, sentPayload, "name")
	})

	t.Run("non-existent namespace -> wrapped error includes AWS error details", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{"__type":"ResourceNotFoundException","message":"rule groups namespace missing-namespace not found"}`)),
				},
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":    "us-east-1",
				"workspace": "ws-abc123",
				"namespace": "missing-namespace",
				"data":      validRulesYAML,
			},
			HTTP:           httpContext,
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
			Integration:    validIntegrationContext(),
		})

		require.ErrorContains(t, err, "failed to update Prometheus rule group namespace")
		require.ErrorContains(t, err, "missing-namespace not found")
	})
}
