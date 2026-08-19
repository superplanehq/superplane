package prometheus

import (
	"encoding/base64"
	"fmt"
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

func Test__GetRuleGroupNamespace__Execute(t *testing.T) {
	component := &GetRuleGroupNamespace{}

	t.Run("valid request -> emits namespace with decoded rules document", func(t *testing.T) {
		encodedData := base64.StdEncoding.EncodeToString([]byte(validRulesYAML))
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(fmt.Sprintf(`{
						"ruleGroupsNamespace": {
							"arn": "arn:aws:aps:us-east-1:123456789012:rulegroupsnamespace/ws-abc123/example-alerts",
							"name": "example-alerts",
							"status": {"statusCode": "ACTIVE"},
							"data": %q,
							"createdAt": 1700000000,
							"modifiedAt": 1700000100
						}
					}`, encodedData))),
				},
			},
		}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":    "us-east-1",
				"workspace": "ws-abc123",
				"namespace": "example-alerts",
			},
			HTTP:           httpContext,
			ExecutionState: execState,
			Integration:    validIntegrationContext(),
		})

		require.NoError(t, err)
		assert.Equal(t, "aws.prometheus.ruleGroupNamespace", execState.Type)

		payload := execState.Payloads[0].(map[string]any)["data"].(map[string]any)
		assert.Equal(t, "ws-abc123", payload["workspaceId"])
		namespace, ok := payload["namespace"].(*RuleGroupsNamespaceDescription)
		require.True(t, ok)
		assert.Equal(t, "example-alerts", namespace.Name)
		assert.Equal(t, "ACTIVE", namespace.Status.StatusCode)
		assert.Equal(t, validRulesYAML, namespace.Data)

		require.Len(t, httpContext.Requests, 1)
		request := httpContext.Requests[0]
		assert.Equal(t, http.MethodGet, request.Method)
		assert.Equal(t, "https://aps.us-east-1.amazonaws.com/workspaces/ws-abc123/rulegroupsnamespaces/example-alerts", request.URL.String())
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

		require.ErrorContains(t, err, "failed to get Prometheus rule group namespace")
		require.ErrorContains(t, err, "missing-namespace not found")
	})

	t.Run("async status transitions -> status code is passed through unchanged", func(t *testing.T) {
		for _, statusCode := range []string{"CREATING", "ACTIVE", "UPDATING", "DELETING", "CREATION_FAILED", "UPDATE_FAILED"} {
			t.Run(statusCode, func(t *testing.T) {
				encodedData := base64.StdEncoding.EncodeToString([]byte(validRulesYAML))
				httpContext := &contexts.HTTPContext{
					Responses: []*http.Response{
						{
							StatusCode: http.StatusOK,
							Body: io.NopCloser(strings.NewReader(fmt.Sprintf(`{
								"ruleGroupsNamespace": {
									"arn": "arn:aws:aps:us-east-1:123456789012:rulegroupsnamespace/ws-abc123/example-alerts",
									"name": "example-alerts",
									"status": {"statusCode": %q},
									"data": %q,
									"createdAt": 1700000000,
									"modifiedAt": 1700000100
								}
							}`, statusCode, encodedData))),
						},
					},
				}

				execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
				err := component.Execute(core.ExecutionContext{
					Configuration: map[string]any{
						"region":    "us-east-1",
						"workspace": "ws-abc123",
						"namespace": "example-alerts",
					},
					HTTP:           httpContext,
					ExecutionState: execState,
					Integration:    validIntegrationContext(),
				})

				require.NoError(t, err)
				payload := execState.Payloads[0].(map[string]any)["data"].(map[string]any)
				namespace := payload["namespace"].(*RuleGroupsNamespaceDescription)
				assert.Equal(t, statusCode, namespace.Status.StatusCode)
			})
		}
	})
}
