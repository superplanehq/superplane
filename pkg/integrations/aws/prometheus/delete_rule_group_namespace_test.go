package prometheus

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

func Test__DeleteRuleGroupNamespace__Setup(t *testing.T) {
	component := &DeleteRuleGroupNamespace{}

	t.Run("missing namespace -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":    "us-east-1",
				"workspace": "ws-abc123",
				"namespace": " ",
			},
		})

		require.ErrorContains(t, err, "namespace is required")
	})

	t.Run("valid configuration -> ok", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":    "us-east-1",
				"workspace": "ws-abc123",
				"namespace": "example-alerts",
			},
		})

		require.NoError(t, err)
	})

	t.Run("valid configuration -> stores workspace alias in metadata", func(t *testing.T) {
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
				"namespace": "example-alerts",
			},
			HTTP:        httpContext,
			Integration: validIntegrationContext(),
			Metadata:    metadata,
		})

		require.NoError(t, err)
		stored, ok := metadata.Get().(WorkspaceNodeMetadata)
		require.True(t, ok)
		assert.Equal(t, "ws-abc123", stored.WorkspaceID)
		assert.Equal(t, "metrics", stored.WorkspaceAlias)
	})
}

func Test__DeleteRuleGroupNamespace__Execute(t *testing.T) {
	component := &DeleteRuleGroupNamespace{}

	t.Run("valid request -> deletes namespace", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusAccepted,
					Body:       io.NopCloser(strings.NewReader("")),
				},
			},
		}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":      "us-east-1",
				"workspace":   "ws-abc123",
				"namespace":   "example-alerts",
				"clientToken": "token-1",
			},
			HTTP:           httpContext,
			ExecutionState: execState,
			Integration:    validIntegrationContext(),
		})

		require.NoError(t, err)
		assert.Equal(t, "aws.prometheus.ruleGroupNamespace.deleted", execState.Type)

		payload := execState.Payloads[0].(map[string]any)["data"].(map[string]any)
		assert.Equal(t, "ws-abc123", payload["workspaceId"])
		assert.Equal(t, "example-alerts", payload["namespace"])
		assert.Equal(t, true, payload["deleted"])

		require.Len(t, httpContext.Requests, 1)
		request := httpContext.Requests[0]
		assert.Equal(t, http.MethodDelete, request.Method)
		assert.Equal(t, "https://aps.us-east-1.amazonaws.com/workspaces/ws-abc123/rulegroupsnamespaces/example-alerts?clientToken=token-1", request.URL.String())
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
			},
			HTTP:           httpContext,
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
			Integration:    validIntegrationContext(),
		})

		require.ErrorContains(t, err, "failed to delete Prometheus rule group namespace")
		require.ErrorContains(t, err, "missing-namespace not found")
	})
}
