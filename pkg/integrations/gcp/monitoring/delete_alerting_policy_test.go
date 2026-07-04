package monitoring

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	gcpcommon "github.com/superplanehq/superplane/pkg/integrations/gcp/common"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__DeleteAlertingPolicy__Execute(t *testing.T) {
	d := &DeleteAlertingPolicy{}

	t.Run("deletes policy -> emits deleted event", func(t *testing.T) {
		var deleteURL string
		mc := &mockClient{
			projectID: "my-project",
			deleteFunc: func(ctx context.Context, url string) ([]byte, error) {
				deleteURL = url
				return []byte("{}"), nil
			},
		}
		withFactory(mc)

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := d.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"alertPolicy": "projects/my-project/alertPolicies/123"},
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.True(t, state.Passed)
		assert.Equal(t, "gcp.monitoring.alertingPolicy.deleted", state.Type)
		assert.True(t, strings.HasSuffix(deleteURL, "/projects/my-project/alertPolicies/123"))
		data := state.Payloads[0].(map[string]any)["data"].(map[string]any)
		assert.Equal(t, "123", data["id"])
	})

	t.Run("cross-project -> fails before API call", func(t *testing.T) {
		var called bool
		mc := &mockClient{
			projectID: "my-project",
			deleteFunc: func(ctx context.Context, url string) ([]byte, error) {
				called = true
				return nil, nil
			},
		}
		withFactory(mc)

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := d.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"alertPolicy": "projects/other/alertPolicies/123"},
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.False(t, state.Passed)
		assert.False(t, called)
		assert.Contains(t, state.FailureMessage, "cross-project")
	})

	t.Run("API error -> fails execution", func(t *testing.T) {
		mc := &mockClient{
			projectID: "my-project",
			deleteFunc: func(ctx context.Context, url string) ([]byte, error) {
				return nil, &gcpcommon.GCPAPIError{StatusCode: http.StatusNotFound, Message: "not found"}
			},
		}
		withFactory(mc)

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := d.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"alertPolicy": "projects/my-project/alertPolicies/123"},
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.False(t, state.Passed)
		assert.Contains(t, state.FailureMessage, "failed to delete alerting policy")
	})
}
