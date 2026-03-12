package clouddns

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	testcontexts "github.com/superplanehq/superplane/test/support/contexts"
)

func TestPollChangeUntilDone(t *testing.T) {
	t.Run("returns early when execution is already finished", func(t *testing.T) {
		err := pollChangeUntilDone(core.ActionContext{
			ExecutionState: &testcontexts.ExecutionStateContext{
				Finished: true,
				KVs:      map[string]string{},
			},
			Metadata: &testcontexts.MetadataContext{},
		})
		require.NoError(t, err)
	})

	t.Run("schedules another poll when change is pending", func(t *testing.T) {
		SetClientFactory(func(_ core.HTTPContext, _ core.IntegrationContext) (Client, error) {
			return &mockClient{
				projectID: "my-project",
				getURL: func(_ context.Context, _ string) ([]byte, error) {
					return json.Marshal(map[string]any{
						"id":        "3",
						"status":    "pending",
						"startTime": "2026-01-28T10:30:00.000Z",
					})
				},
			}, nil
		})

		state := &testcontexts.ExecutionStateContext{KVs: map[string]string{}}
		requests := &testcontexts.RequestContext{}
		err := pollChangeUntilDone(core.ActionContext{
			ExecutionState: state,
			Requests:       requests,
			Metadata: &testcontexts.MetadataContext{Metadata: RecordSetPollMetadata{
				ChangeID:    "3",
				ManagedZone: "my-zone",
				RecordName:  "api.example.com.",
				RecordType:  "A",
				StartTime:   "2026-01-28T10:30:00.000Z",
			}},
		})

		require.NoError(t, err)
		assert.Equal(t, pollChangeActionName, requests.Action)
		assert.Equal(t, pollInterval, requests.Duration)
		assert.False(t, state.IsFinished())
	})

	t.Run("emits output when change is done", func(t *testing.T) {
		SetClientFactory(func(_ core.HTTPContext, _ core.IntegrationContext) (Client, error) {
			return &mockClient{
				projectID: "my-project",
				getURL: func(_ context.Context, _ string) ([]byte, error) {
					return json.Marshal(map[string]any{
						"id":        "3",
						"status":    "done",
						"startTime": "2026-01-28T10:30:00.000Z",
					})
				},
			}, nil
		})

		state := &testcontexts.ExecutionStateContext{KVs: map[string]string{}}
		err := pollChangeUntilDone(core.ActionContext{
			ExecutionState: state,
			Metadata: &testcontexts.MetadataContext{Metadata: RecordSetPollMetadata{
				ChangeID:    "3",
				ManagedZone: "my-zone",
				RecordName:  "api.example.com.",
				RecordType:  "A",
				StartTime:   "2026-01-28T10:30:00.000Z",
			}},
		})

		require.NoError(t, err)
		assert.True(t, state.Finished)
		assert.True(t, state.Passed)
		require.Len(t, state.Payloads, 1)
		wrapper := state.Payloads[0].(map[string]any)
		data := wrapper["data"].(map[string]any)
		change := data["change"].(map[string]any)
		record := data["record"].(map[string]any)
		assert.Equal(t, "done", change["status"])
		assert.Equal(t, "api.example.com.", record["name"])
		assert.Equal(t, "A", record["type"])
	})

	t.Run("fails when change status is unexpected", func(t *testing.T) {
		SetClientFactory(func(_ core.HTTPContext, _ core.IntegrationContext) (Client, error) {
			return &mockClient{
				projectID: "my-project",
				getURL: func(_ context.Context, _ string) ([]byte, error) {
					return json.Marshal(map[string]any{
						"id":        "3",
						"status":    "failed",
						"startTime": "2026-01-28T10:30:00.000Z",
					})
				},
			}, nil
		})

		state := &testcontexts.ExecutionStateContext{KVs: map[string]string{}}
		requests := &testcontexts.RequestContext{}
		err := pollChangeUntilDone(core.ActionContext{
			ExecutionState: state,
			Requests:       requests,
			Metadata: &testcontexts.MetadataContext{Metadata: RecordSetPollMetadata{
				ChangeID:    "3",
				ManagedZone: "my-zone",
				RecordName:  "api.example.com.",
				RecordType:  "A",
				StartTime:   "2026-01-28T10:30:00.000Z",
			}},
		})

		require.NoError(t, err)
		assert.True(t, state.Finished)
		assert.False(t, state.Passed)
		assert.Contains(t, state.FailureMessage, "unexpected Cloud DNS change status")
		assert.Empty(t, requests.Action)
	})
}
