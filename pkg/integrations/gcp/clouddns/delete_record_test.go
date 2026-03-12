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

func TestDeleteRecord_Metadata(t *testing.T) {
	c := &DeleteRecord{}
	assert.Equal(t, "gcp.clouddns.deleteRecord", c.Name())
	assert.Equal(t, "Cloud DNS • Delete Record", c.Label())
	assert.NotEmpty(t, c.Description())
	assert.Equal(t, "gcp", c.Icon())
}

func TestDeleteRecord_ExampleOutput(t *testing.T) {
	c := &DeleteRecord{}
	output := c.ExampleOutput()
	assert.NotEmpty(t, output["type"])
	assert.NotEmpty(t, output["data"])
}

func TestDeleteRecord_Setup(t *testing.T) {
	c := &DeleteRecord{}

	t.Run("succeeds with valid config", func(t *testing.T) {
		err := c.Setup(core.SetupContext{
			Configuration: map[string]any{
				"managedZone": "my-zone",
				"name":        "api.example.com",
				"type":        "A",
			},
			Metadata: &testcontexts.MetadataContext{},
		})
		require.NoError(t, err)
	})

	t.Run("fails when managed zone is missing", func(t *testing.T) {
		err := c.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name": "api.example.com",
				"type": "A",
			},
			Metadata: &testcontexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "managed zone is required")
	})

	t.Run("fails when record name is missing", func(t *testing.T) {
		err := c.Setup(core.SetupContext{
			Configuration: map[string]any{
				"managedZone": "my-zone",
				"type":        "A",
			},
			Metadata: &testcontexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "record name is required")
	})
}

func TestDeleteRecord_Execute(t *testing.T) {
	existingRecord := map[string]any{
		"rrsets": []any{
			map[string]any{
				"name":    "api.example.com.",
				"type":    "A",
				"ttl":     float64(300),
				"rrdatas": []any{"1.2.3.4"},
			},
		},
	}

	t.Run("looks up existing record and emits output when done", func(t *testing.T) {
		SetClientFactory(func(_ core.HTTPContext, _ core.IntegrationContext) (Client, error) {
			return &mockClient{
				projectID: "my-project",
				getURL: func(_ context.Context, _ string) ([]byte, error) {
					return json.Marshal(existingRecord)
				},
				postURL: func(_ context.Context, _ string, _ any) ([]byte, error) {
					return json.Marshal(map[string]any{
						"id":        "5",
						"status":    "done",
						"startTime": "2026-01-28T10:30:00.000Z",
					})
				},
			}, nil
		})

		state := &testcontexts.ExecutionStateContext{KVs: map[string]string{}}
		err := (&DeleteRecord{}).Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"managedZone": "my-zone",
				"name":        "api.example.com",
				"type":        "A",
			},
			ExecutionState: state,
			Metadata:       &testcontexts.MetadataContext{},
		})

		require.NoError(t, err)
		assert.True(t, state.Finished)
		assert.True(t, state.Passed)
	})

	t.Run("fails when record does not exist", func(t *testing.T) {
		SetClientFactory(func(_ core.HTTPContext, _ core.IntegrationContext) (Client, error) {
			return &mockClient{
				projectID: "my-project",
				getURL: func(_ context.Context, _ string) ([]byte, error) {
					return json.Marshal(map[string]any{"rrsets": []any{}})
				},
			}, nil
		})

		state := &testcontexts.ExecutionStateContext{KVs: map[string]string{}}
		err := (&DeleteRecord{}).Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"managedZone": "my-zone",
				"name":        "api.example.com",
				"type":        "A",
			},
			ExecutionState: state,
			Metadata:       &testcontexts.MetadataContext{},
		})

		require.NoError(t, err)
		assert.True(t, state.Finished)
		assert.False(t, state.Passed)
		assert.Contains(t, state.FailureMessage, "not found")
	})
}
