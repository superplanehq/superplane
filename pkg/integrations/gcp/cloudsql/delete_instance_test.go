package cloudsql

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

func Test__DeleteInstance__Setup(t *testing.T) {
	d := &DeleteInstance{}
	setup := func(cfg map[string]any) error {
		return d.Setup(core.SetupContext{Configuration: cfg, Metadata: &contexts.MetadataContext{}})
	}

	t.Run("missing instance -> error", func(t *testing.T) {
		require.ErrorContains(t, setup(map[string]any{}), "instance is required")
	})

	t.Run("valid -> ok", func(t *testing.T) {
		require.NoError(t, setup(map[string]any{"instance": "my-instance"}))
	})
}

func Test__DeleteInstance__Execute(t *testing.T) {
	d := &DeleteInstance{}

	t.Run("confirms deletion when the instance is already gone", func(t *testing.T) {
		mc := &mockClient{
			projectID: "my-project",
			deleteFunc: func(ctx context.Context, url string) ([]byte, error) {
				return nil, &gcpcommon.GCPAPIError{StatusCode: http.StatusNotFound, Message: "not found"}
			},
		}
		withFactory(mc)

		requests := &contexts.RequestContext{}
		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := d.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"instance": "my-instance"},
			Metadata:       &contexts.MetadataContext{},
			Requests:       requests,
			ExecutionState: state,
		})
		require.NoError(t, err)
		// Already-gone instances emit the deletion confirmation immediately,
		// without scheduling a poll.
		assert.True(t, state.Passed)
		assert.Empty(t, requests.Action)
		data := firstData(t, state)
		assert.Equal(t, "my-instance", data["name"])
		assert.Equal(t, true, data["deleted"])
	})

	t.Run("starts deletion and schedules a poll that emits once the instance is gone", func(t *testing.T) {
		var deleteURL string
		stillExists := true
		mc := &mockClient{
			projectID: "my-project",
			deleteFunc: func(ctx context.Context, url string) ([]byte, error) {
				deleteURL = url
				return []byte(`{"name":"op-456","status":"PENDING","targetId":"my-instance"}`), nil
			},
			getFunc: func(ctx context.Context, url string) ([]byte, error) {
				if stillExists {
					return []byte(`{"name":"my-instance","state":"PENDING_DELETE"}`), nil
				}
				return nil, &gcpcommon.GCPAPIError{StatusCode: http.StatusNotFound, Message: "not found"}
			},
		}
		withFactory(mc)

		metadata := &contexts.MetadataContext{}
		requests := &contexts.RequestContext{}
		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := d.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"instance": "my-instance"},
			Metadata:       metadata,
			Requests:       requests,
			ExecutionState: state,
		})
		require.NoError(t, err)
		// Execute starts the delete and schedules a poll rather than emitting.
		assert.Equal(t, pollHookName, requests.Action)
		assert.False(t, state.Passed)
		assert.True(t, strings.HasSuffix(deleteURL, "/projects/my-project/instances/my-instance"))

		// First poll: instance still present -> re-schedules, no emit.
		reqs := &contexts.RequestContext{}
		require.NoError(t, d.HandleHook(core.ActionHookContext{Name: pollHookName, Metadata: metadata, Requests: reqs, ExecutionState: state}))
		assert.Equal(t, pollHookName, reqs.Action)
		assert.Empty(t, state.Payloads)

		// Next poll: instance gone (404) -> emits the deletion confirmation.
		stillExists = false
		require.NoError(t, d.HandleHook(core.ActionHookContext{Name: pollHookName, Metadata: metadata, Requests: &contexts.RequestContext{}, ExecutionState: state}))
		assert.True(t, state.Passed)
		assert.Equal(t, "gcp.cloudsql.instance", state.Type)
		data := firstData(t, state)
		assert.Equal(t, "my-instance", data["name"])
		assert.Equal(t, true, data["deleted"])
	})
}
