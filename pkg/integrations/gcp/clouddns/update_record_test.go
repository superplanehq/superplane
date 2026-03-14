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

func TestUpdateRecord_Metadata(t *testing.T) {
	c := &UpdateRecord{}
	assert.Equal(t, "gcp.clouddns.updateRecord", c.Name())
	assert.Equal(t, "Cloud DNS • Update Record", c.Label())
	assert.NotEmpty(t, c.Description())
	assert.Equal(t, "gcp", c.Icon())
}

func TestUpdateRecord_ExampleOutput(t *testing.T) {
	c := &UpdateRecord{}
	output := c.ExampleOutput()
	assert.NotEmpty(t, output["type"])
	assert.NotEmpty(t, output["data"])
}

func TestUpdateRecord_Setup(t *testing.T) {
	c := &UpdateRecord{}

	t.Run("succeeds with valid config", func(t *testing.T) {
		err := c.Setup(core.SetupContext{
			Configuration: map[string]any{
				"managedZone": "my-zone",
				"name":        "api.example.com",
				"type":        "A",
				"ttl":         300,
				"rrdatas":     []string{"5.6.7.8"},
			},
			Metadata: &testcontexts.MetadataContext{},
		})
		require.NoError(t, err)
	})

	t.Run("fails when rrdatas is empty", func(t *testing.T) {
		err := c.Setup(core.SetupContext{
			Configuration: map[string]any{
				"managedZone": "my-zone",
				"name":        "api.example.com",
				"type":        "A",
				"rrdatas":     []string{},
			},
			Metadata: &testcontexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "at least one record value is required")
	})
}

func TestUpdateRecord_Execute(t *testing.T) {
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

	t.Run("deletes old record and creates new one", func(t *testing.T) {
		var capturedBody any
		SetClientFactory(func(_ core.HTTPContext, _ core.IntegrationContext) (Client, error) {
			return &mockClient{
				projectID: "my-project",
				getURL: func(_ context.Context, _ string) ([]byte, error) {
					return json.Marshal(existingRecord)
				},
				postURL: func(_ context.Context, _ string, body any) ([]byte, error) {
					capturedBody = body
					return json.Marshal(map[string]any{
						"id":        "10",
						"status":    "done",
						"startTime": "2026-01-28T10:30:00.000Z",
					})
				},
			}, nil
		})

		state := &testcontexts.ExecutionStateContext{KVs: map[string]string{}}
		err := (&UpdateRecord{}).Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"managedZone": "my-zone",
				"name":        "api.example.com",
				"type":        "A",
				"ttl":         300,
				"rrdatas":     []string{"5.6.7.8"},
			},
			ExecutionState: state,
			Metadata:       &testcontexts.MetadataContext{},
		})

		require.NoError(t, err)
		assert.True(t, state.Finished)
		assert.True(t, state.Passed)

		bodyMap := capturedBody.(map[string]any)
		additions := bodyMap["additions"].([]ResourceRecordSet)
		deletions := bodyMap["deletions"].([]ResourceRecordSet)
		assert.Equal(t, []string{"5.6.7.8"}, additions[0].Rrdatas)
		assert.Equal(t, []string{"1.2.3.4"}, deletions[0].Rrdatas)
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
		err := (&UpdateRecord{}).Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"managedZone": "my-zone",
				"name":        "api.example.com",
				"type":        "A",
				"rrdatas":     []string{"5.6.7.8"},
			},
			ExecutionState: state,
			Metadata:       &testcontexts.MetadataContext{},
		})

		require.NoError(t, err)
		assert.False(t, state.Passed)
		assert.Contains(t, state.FailureMessage, "not found")
	})

	t.Run("fails when change status is unexpected", func(t *testing.T) {
		SetClientFactory(func(_ core.HTTPContext, _ core.IntegrationContext) (Client, error) {
			return &mockClient{
				projectID: "my-project",
				getURL: func(_ context.Context, _ string) ([]byte, error) {
					return json.Marshal(existingRecord)
				},
				postURL: func(_ context.Context, _ string, _ any) ([]byte, error) {
					return json.Marshal(map[string]any{
						"id":        "11",
						"status":    "failed",
						"startTime": "2026-01-28T10:30:00.000Z",
					})
				},
			}, nil
		})

		state := &testcontexts.ExecutionStateContext{KVs: map[string]string{}}
		requests := &testcontexts.RequestContext{}
		err := (&UpdateRecord{}).Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"managedZone": "my-zone",
				"name":        "api.example.com",
				"type":        "A",
				"ttl":         300,
				"rrdatas":     []string{"5.6.7.8"},
			},
			ExecutionState: state,
			Metadata:       &testcontexts.MetadataContext{},
			Requests:       requests,
		})

		require.NoError(t, err)
		assert.True(t, state.Finished)
		assert.False(t, state.Passed)
		assert.Contains(t, state.FailureMessage, "unexpected Cloud DNS change status")
		assert.Empty(t, requests.Action)
	})
}
