package sentry

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func Test__UpdateIssue__Setup(t *testing.T) {
	c := &UpdateIssue{}

	t.Run("missing organization -> error", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{"issueId": "123", "status": "resolved"},
			Metadata:      &contexts.MetadataContext{},
		}
		err := c.Setup(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "organization")
	})

	t.Run("missing issueId -> error", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{"organization": "my-org", "status": "resolved"},
			Metadata:      &contexts.MetadataContext{},
		}
		err := c.Setup(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "issueId")
	})

	t.Run("missing status and assignedTo -> error", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{"organization": "my-org", "issueId": "123"},
			Metadata:      &contexts.MetadataContext{},
		}
		err := c.Setup(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "status or assignedTo")
	})

	t.Run("valid with status -> success", func(t *testing.T) {
		meta := &contexts.MetadataContext{}
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"organization": "my-org",
				"issueId":      "123",
				"status":       "resolved",
			},
			Metadata: meta,
		}
		err := c.Setup(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, meta.Get())
	})

	t.Run("valid with assignedTo -> success", func(t *testing.T) {
		meta := &contexts.MetadataContext{}
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"organization": "my-org",
				"issueId":      "123",
				"assignedTo":   "user@example.com",
			},
			Metadata: meta,
		}
		err := c.Setup(ctx)
		assert.NoError(t, err)
	})
}

func Test__UpdateIssue__Execute(t *testing.T) {
	c := &UpdateIssue{}

	t.Run("decode config error -> error", func(t *testing.T) {
		ctx := core.ExecutionContext{
			Configuration:  "not-a-map",
			Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"authToken": "t", "baseURL": "https://sentry.io"}},
			ExecutionState: &contexts.ExecutionStateContext{},
		}
		err := c.Execute(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "decode")
	})

	t.Run("success -> emits output", func(t *testing.T) {
		respBody := `{"id":"123","shortId":"ABC-1","status":"resolved","title":"Error"}`
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(respBody))},
			},
		}
		integration := &contexts.IntegrationContext{
			Configuration: map[string]any{"authToken": "token", "baseURL": "https://sentry.io"},
		}
		execState := &contexts.ExecutionStateContext{}
		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"organization": "my-org",
				"issueId":      "123",
				"status":       "resolved",
			},
			HTTP:           httpCtx,
			Integration:    integration,
			ExecutionState: execState,
		}
		err := c.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, execState.Finished)
		assert.True(t, execState.Passed)
		assert.Equal(t, "sentry.issue", execState.Type)
		assert.Len(t, execState.Payloads, 1)
	})
}
