package cloudfunctions

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	testcontexts "github.com/superplanehq/superplane/test/support/contexts"
)

func TestInvokeFunction_Metadata(t *testing.T) {
	c := &InvokeFunction{}
	assert.Equal(t, "gcp.cloudfunctions.invokeFunction", c.Name())
	assert.Equal(t, "Cloud Functions • Invoke Function", c.Label())
	assert.NotEmpty(t, c.Description())
	assert.NotEmpty(t, c.Documentation())
	assert.Equal(t, "gcp", c.Icon())
	assert.Equal(t, "gray", c.Color())
	assert.Nil(t, c.Actions())
}

func TestInvokeFunction_ExampleOutput(t *testing.T) {
	c := &InvokeFunction{}
	output := c.ExampleOutput()
	assert.NotEmpty(t, output["type"])
	assert.NotEmpty(t, output["timestamp"])
	payload, ok := output["data"].(map[string]any)
	require.True(t, ok)
	assert.NotEmpty(t, payload["functionName"])
	assert.NotEmpty(t, payload["executionId"])
}

func TestInvokeFunction_Setup(t *testing.T) {
	c := &InvokeFunction{}

	t.Run("stores function name in metadata", func(t *testing.T) {
		meta := &testcontexts.MetadataContext{}
		err := c.Setup(core.SetupContext{
			Configuration: map[string]any{
				"location": "us-central1",
				"function": "projects/my-project/locations/us-central1/functions/hello-world",
			},
			Metadata: meta,
		})
		require.NoError(t, err)
		stored := meta.Get().(InvokeFunctionMetadata)
		assert.Equal(t, "projects/my-project/locations/us-central1/functions/hello-world", stored.FunctionName)
	})

	t.Run("returns error when location is missing", func(t *testing.T) {
		err := c.Setup(core.SetupContext{
			Configuration: map[string]any{
				"function": "projects/my-project/locations/us-central1/functions/hello-world",
			},
			Metadata: &testcontexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "location is required")
	})

	t.Run("returns error when function is missing", func(t *testing.T) {
		err := c.Setup(core.SetupContext{
			Configuration: map[string]any{"location": "us-central1"},
			Metadata:      &testcontexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "function is required")
	})
}

func TestInvokeFunction_Execute(t *testing.T) {
	t.Run("invokes gen1 function via :call API and emits parsed JSON result", func(t *testing.T) {
		functionName := "projects/my-project/locations/us-central1/functions/hello-world"
		SetClientFactory(func(_ core.HTTPContext, _ core.IntegrationContext) (Client, error) {
			return &mockClient{
				projectID: "my-project",
				postURL: func(_ context.Context, fullURL string, body any) ([]byte, error) {
					assert.Equal(t,
						"https://cloudfunctions.googleapis.com/v1/"+functionName+":call",
						fullURL,
					)
					bodyMap := body.(map[string]any)
					assert.NotEmpty(t, bodyMap["data"])
					return json.Marshal(map[string]any{
						"executionId": "exec-123",
						"result":      `{"message":"hello"}`,
					})
				},
			}, nil
		})

		state := &testcontexts.ExecutionStateContext{KVs: map[string]string{}}
		err := (&InvokeFunction{}).Execute(core.ExecutionContext{
			Configuration:  map[string]any{"location": "us-central1", "function": functionName},
			ExecutionState: state,
			NodeMetadata:   &testcontexts.MetadataContext{Metadata: InvokeFunctionMetadata{FunctionName: functionName}},
		})

		require.NoError(t, err)
		assert.True(t, state.Finished)
		assert.True(t, state.Passed)
		require.Len(t, state.Payloads, 1)
		wrapper := state.Payloads[0].(map[string]any)
		data := wrapper["data"].(map[string]any)
		assert.Equal(t, functionName, data["functionName"])
		assert.Equal(t, "exec-123", data["executionId"])
		result, ok := data["result"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "hello", result["message"])
	})

	t.Run("invokes gen2 function via HTTP trigger URI", func(t *testing.T) {
		functionName := "projects/my-project/locations/us-central1/functions/hello-world"
		triggerURI := "https://hello-world-abc123-uc.a.run.app"
		SetClientFactory(func(_ core.HTTPContext, _ core.IntegrationContext) (Client, error) {
			return &mockClient{
				projectID: "my-project",
				postURL: func(_ context.Context, fullURL string, body any) ([]byte, error) {
					assert.Equal(t, triggerURI, fullURL)
					return json.Marshal(map[string]any{"message": "hello gen2"})
				},
			}, nil
		})

		state := &testcontexts.ExecutionStateContext{KVs: map[string]string{}}
		err := (&InvokeFunction{}).Execute(core.ExecutionContext{
			Configuration:  map[string]any{"location": "us-central1", "function": functionName},
			ExecutionState: state,
			NodeMetadata: &testcontexts.MetadataContext{Metadata: InvokeFunctionMetadata{
				FunctionName: functionName,
				Environment:  "GEN_2",
				FunctionURI:  triggerURI,
			}},
		})

		require.NoError(t, err)
		assert.True(t, state.Finished)
		assert.True(t, state.Passed)
		require.Len(t, state.Payloads, 1)
		wrapper := state.Payloads[0].(map[string]any)
		data := wrapper["data"].(map[string]any)
		assert.Equal(t, functionName, data["functionName"])
		result, ok := data["result"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "hello gen2", result["message"])
	})

	t.Run("stores raw string when result is not JSON", func(t *testing.T) {
		functionName := "projects/p/locations/l/functions/f"
		SetClientFactory(func(_ core.HTTPContext, _ core.IntegrationContext) (Client, error) {
			return &mockClient{
				projectID: "my-project",
				postURL: func(_ context.Context, _ string, _ any) ([]byte, error) {
					return json.Marshal(map[string]any{
						"executionId": "exec-789",
						"result":      "plain text response",
					})
				},
			}, nil
		})

		state := &testcontexts.ExecutionStateContext{KVs: map[string]string{}}
		err := (&InvokeFunction{}).Execute(core.ExecutionContext{
			Configuration:  map[string]any{"location": "us-central1", "function": functionName},
			ExecutionState: state,
			NodeMetadata:   &testcontexts.MetadataContext{Metadata: InvokeFunctionMetadata{FunctionName: functionName}},
		})

		require.NoError(t, err)
		require.Len(t, state.Payloads, 1)
		wrapper := state.Payloads[0].(map[string]any)
		data := wrapper["data"].(map[string]any)
		assert.Equal(t, "plain text response", data["resultRaw"])
		assert.Nil(t, data["result"])
	})

	t.Run("fails when function returns an error field", func(t *testing.T) {
		functionName := "projects/p/locations/l/functions/broken"
		SetClientFactory(func(_ core.HTTPContext, _ core.IntegrationContext) (Client, error) {
			return &mockClient{
				projectID: "my-project",
				postURL: func(_ context.Context, _ string, _ any) ([]byte, error) {
					return json.Marshal(map[string]any{
						"executionId": "exec-456",
						"error":       "function panicked",
					})
				},
			}, nil
		})

		state := &testcontexts.ExecutionStateContext{KVs: map[string]string{}}
		err := (&InvokeFunction{}).Execute(core.ExecutionContext{
			Configuration:  map[string]any{"location": "us-central1", "function": functionName},
			ExecutionState: state,
			NodeMetadata:   &testcontexts.MetadataContext{Metadata: InvokeFunctionMetadata{FunctionName: functionName}},
		})

		require.NoError(t, err)
		assert.True(t, state.Finished)
		assert.False(t, state.Passed)
		assert.Contains(t, state.FailureMessage, "function panicked")
	})
}
