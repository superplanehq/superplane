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
				"namespace": "application-rules",
			},
		})

		require.NoError(t, err)
	})
}

func Test__DeleteRuleGroupNamespace__Execute(t *testing.T) {
	component := &DeleteRuleGroupNamespace{}

	t.Run("valid request -> emits delete result", func(t *testing.T) {
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
				"namespace":   "application-rules",
				"clientToken": "token-1",
			},
			HTTP:           httpContext,
			ExecutionState: execState,
			Integration:    validIntegrationContext(),
			NodeMetadata: &contexts.MetadataContext{Metadata: RuleGroupsNamespaceNodeMetadata{
				WorkspaceAlias: "metrics",
			}},
		})

		require.NoError(t, err)
		assert.Equal(t, "aws.prometheus.ruleGroupNamespace.deleted", execState.Type)

		payload := execState.Payloads[0].(map[string]any)["data"].(map[string]any)
		assert.Equal(t, "ws-abc123", payload["workspaceId"])
		assert.Equal(t, "metrics", payload["workspaceAlias"])
		assert.Equal(t, "application-rules", payload["namespace"])
		assert.Equal(t, true, payload["deleted"])

		require.Len(t, httpContext.Requests, 1)
		request := httpContext.Requests[0]
		assert.Equal(t, http.MethodDelete, request.Method)
		assert.Equal(t, "https://aps.us-east-1.amazonaws.com/workspaces/ws-abc123/rulegroupsnamespaces/application-rules?clientToken=token-1", request.URL.String())
	})
}
