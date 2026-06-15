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

func Test__GetRuleGroupNamespace__Setup(t *testing.T) {
	component := &GetRuleGroupNamespace{}

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
				"namespace": "application-rules",
			},
			HTTP:        httpContext,
			Integration: validIntegrationContext(),
			Metadata:    metadata,
		})

		require.NoError(t, err)
		stored, ok := metadata.Get().(RuleGroupsNamespaceNodeMetadata)
		require.True(t, ok)
		assert.Equal(t, "metrics", stored.WorkspaceAlias)
		assert.Equal(t, "application-rules", stored.Namespace)
	})
}

func Test__GetRuleGroupNamespace__Execute(t *testing.T) {
	component := &GetRuleGroupNamespace{}

	t.Run("valid request -> emits namespace", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"ruleGroupsNamespace": {
							"arn": "arn:aws:aps:us-east-1:123456789012:rulegroupsnamespace/ws-abc123/application-rules",
							"createdAt": 1717846800,
							"data": "Z3JvdXBzOiBbXQ==",
							"modifiedAt": 1717847100,
							"name": "application-rules",
							"status": {"statusCode": "ACTIVE"},
							"tags": {"env": "prod"}
						}
					}`)),
				},
			},
		}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":    "us-east-1",
				"workspace": "ws-abc123",
				"namespace": "application-rules",
			},
			HTTP:           httpContext,
			ExecutionState: execState,
			Integration:    validIntegrationContext(),
		})

		require.NoError(t, err)
		assert.Equal(t, "aws.prometheus.ruleGroupNamespace", execState.Type)

		payload := execState.Payloads[0].(map[string]any)["data"].(map[string]any)
		namespace, ok := payload["ruleGroupNamespace"].(*RuleGroupsNamespaceDescription)
		require.True(t, ok)
		assert.Equal(t, "application-rules", namespace.Name)
		assert.Equal(t, "ACTIVE", namespace.Status.StatusCode)
		assert.Equal(t, "Z3JvdXBzOiBbXQ==", namespace.Data)
		assert.Equal(t, "ws-abc123", payload["workspaceId"])

		require.Len(t, httpContext.Requests, 1)
		request := httpContext.Requests[0]
		assert.Equal(t, http.MethodGet, request.Method)
		assert.Equal(t, "https://aps.us-east-1.amazonaws.com/workspaces/ws-abc123/rulegroupsnamespaces/application-rules", request.URL.String())
	})
}
