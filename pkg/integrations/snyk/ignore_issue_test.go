package snyk

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

func TestIgnoreIssueComponent(t *testing.T) {
	component := &IgnoreIssue{}

	assert.Equal(t, "snyk.ignoreIssue", component.Name())
	assert.Equal(t, "Ignore Issue", component.Label())
	assert.Equal(t, "Ignore a specific Snyk security issue", component.Description())
	assert.Equal(t, "shield", component.Icon())

	configFields := component.Configuration()
	assert.Len(t, configFields, 4)

	fieldNames := make(map[string]bool)
	for _, field := range configFields {
		fieldNames[field.Name] = true
	}

	expectedFields := []string{"projectId", "issueId", "reason", "expiresAt"}
	for _, fieldName := range expectedFields {
		assert.True(t, fieldNames[fieldName], "Missing field: %s", fieldName)
	}
}

func TestIgnoreIssueExampleOutput(t *testing.T) {
	component := &IgnoreIssue{}
	example := component.ExampleOutput()

	assert.NotNil(t, example)
}

func Test__IgnoreIssue__Setup(t *testing.T) {
	component := &IgnoreIssue{}

	tests := []struct {
		name          string
		configuration map[string]any
		expectError   bool
		errorContains string
	}{
		{
			name:          "missing projectId",
			configuration: map[string]any{"issueId": "i1", "reason": "test"},
			expectError:   true,
			errorContains: "projectId is required",
		},
		{
			name:          "missing issueId",
			configuration: map[string]any{"projectId": "p1", "reason": "test"},
			expectError:   true,
			errorContains: "issueId is required",
		},
		{
			name:          "missing reason",
			configuration: map[string]any{"projectId": "p1", "issueId": "i1"},
			expectError:   true,
			errorContains: "reason is required",
		},
		{
			name: "valid configuration",
			configuration: map[string]any{
				"projectId": "p1",
				"issueId":   "SNYK-JS-123",
				"reason":    "false positive",
			},
			expectError: false,
		},
		{
			name: "valid configuration with expiresAt",
			configuration: map[string]any{
				"projectId": "p1",
				"issueId":   "SNYK-JS-123",
				"reason":    "temporary acceptance",
				"expiresAt": "2026-06-01T00:00:00Z",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metadata := &contexts.MetadataContext{}
			ctx := core.SetupContext{
				Configuration: tt.configuration,
				Metadata:      metadata,
			}

			err := component.Setup(ctx)
			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, metadata.Metadata)
			}
		})
	}
}

func Test__IgnoreIssue__Execute(t *testing.T) {
	component := &IgnoreIssue{}

	t.Run("successful ignore", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"ok": true}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken":       "test-token",
				"organizationId": "org-123",
			},
		}

		execState := &contexts.ExecutionStateContext{
			KVs: map[string]string{},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"projectId": "proj-456",
				"issueId":   "SNYK-JS-789",
				"reason":    "false positive",
			},
			HTTP:           httpCtx,
			Integration:    integrationCtx,
			ExecutionState: execState,
		})

		require.NoError(t, err)
		assert.True(t, execState.Finished)
		assert.True(t, execState.Passed)
		assert.Equal(t, "snyk.issue.ignored", execState.Type)
		assert.Equal(t, core.DefaultOutputChannel.Name, execState.Channel)
		require.Len(t, execState.Payloads, 1)

		// Verify the HTTP request was made correctly
		require.Len(t, httpCtx.Requests, 1)
		req := httpCtx.Requests[0]
		assert.Equal(t, "POST", req.Method)
		assert.Contains(t, req.URL.String(), "/v1/org/org-123/project/proj-456/ignore/SNYK-JS-789")
	})

	t.Run("API error", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusForbidden,
					Body:       io.NopCloser(strings.NewReader(`{"error": "forbidden"}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken":       "test-token",
				"organizationId": "org-123",
			},
		}

		execState := &contexts.ExecutionStateContext{
			KVs: map[string]string{},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"projectId": "proj-456",
				"issueId":   "SNYK-JS-789",
				"reason":    "false positive",
			},
			HTTP:           httpCtx,
			Integration:    integrationCtx,
			ExecutionState: execState,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to ignore issue")
		assert.False(t, execState.Finished)
	})

	t.Run("missing API token", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"organizationId": "org-123",
			},
		}

		execState := &contexts.ExecutionStateContext{
			KVs: map[string]string{},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"projectId": "proj-456",
				"issueId":   "SNYK-JS-789",
				"reason":    "false positive",
			},
			HTTP:           httpCtx,
			Integration:    integrationCtx,
			ExecutionState: execState,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create Snyk client")
	})
}
