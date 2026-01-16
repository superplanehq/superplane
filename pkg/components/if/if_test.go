package ifp

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestIf_Execute_EmitsEmptyEvents(t *testing.T) {
	tests := []struct {
		name            string
		configuration   map[string]any
		inputData       any
		expectedChannel string
	}{
		{
			name:            "if with true condition emits empty event",
			configuration:   map[string]any{"expression": "true"},
			inputData:       map[string]any{"test": "value"},
			expectedChannel: "true",
		},
		{
			name:            "if with false condition emits empty event",
			configuration:   map[string]any{"expression": "false"},
			inputData:       map[string]any{"test": "value"},
			expectedChannel: "false",
		},
		{
			name:            "if with complex true condition emits empty event",
			configuration:   map[string]any{"expression": "$.test == 'value'"},
			inputData:       map[string]any{"test": "value"},
			expectedChannel: "true",
		},
		{
			name:            "if with complex false condition emits empty event",
			configuration:   map[string]any{"expression": "$.test == 'different'"},
			inputData:       map[string]any{"test": "value"},
			expectedChannel: "false",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			ifComponent := &If{}

			stateCtx := &contexts.ExecutionStateContext{}
			metadataCtx := &contexts.MetadataContext{}

			ctx := core.ExecutionContext{
				Data:           tt.inputData,
				Configuration:  tt.configuration,
				ExecutionState: stateCtx,
				Metadata:       metadataCtx,
			}

			err := ifComponent.Execute(ctx)

			assert.NoError(t, err)
			assert.True(t, stateCtx.Passed)
			assert.True(t, stateCtx.Finished)

			// Verify that the expression is stored in metadata
			assert.NotNil(t, metadataCtx.Metadata)
			metadata, ok := metadataCtx.Metadata.(map[string]any)
			assert.True(t, ok)
			assert.Equal(t, tt.configuration["expression"], metadata["expression"])

			assert.Equal(t, tt.expectedChannel, stateCtx.Channel)
			assert.Equal(t, "if.executed", stateCtx.Type)
			assert.Len(t, stateCtx.Payloads, 1)

			// Verify payload structure follows SuperPlane conventions
			payload, ok := stateCtx.Payloads[0].(map[string]any)
			assert.True(t, ok, "payload should be a map")
			assert.Equal(t, "if.executed", payload["type"])
			assert.NotEmpty(t, payload["timestamp"])
			assert.Contains(t, payload, "data")
			data, ok := payload["data"].(map[string]any)
			assert.True(t, ok, "data should be a map")
			assert.Empty(t, data, "data should be empty")
		})
	}
}

func TestIf_Execute_InvalidExpression_ShouldReturnError(t *testing.T) {
	ifComponent := &If{}

	stateCtx := &contexts.ExecutionStateContext{}
	metadataCtx := &contexts.MetadataContext{}

	ctx := core.ExecutionContext{
		Data:           map[string]any{"test": "value"},
		Configuration:  map[string]any{"expression": "invalid expression syntax +++"},
		ExecutionState: stateCtx,
		Metadata:       metadataCtx,
	}

	err := ifComponent.Execute(ctx)
	assert.Error(t, err)

}

func TestIf_Execute_NonBooleanResult_ShouldReturnError(t *testing.T) {
	ifComponent := &If{}

	stateCtx := &contexts.ExecutionStateContext{}
	metadataCtx := &contexts.MetadataContext{}

	ctx := core.ExecutionContext{
		Data:           map[string]any{"test": "value"},
		Configuration:  map[string]any{"expression": "$.test"},
		ExecutionState: stateCtx,
		Metadata:       metadataCtx,
	}

	err := ifComponent.Execute(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid operation: bool(string)")
}

func TestIf_Execute_BothTrueAndFalsePathsEmitEmpty(t *testing.T) {
	tests := []struct {
		name            string
		configuration   map[string]any
		expectedChannel string
	}{
		{
			name:            "true condition previously went to true channel, now emits empty",
			configuration:   map[string]any{"expression": "true"},
			expectedChannel: "true",
		},
		{
			name:            "false condition previously went to false channel, now emits empty",
			configuration:   map[string]any{"expression": "false"},
			expectedChannel: "false",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ifComponent := &If{}

			stateCtx := &contexts.ExecutionStateContext{}
			metadataCtx := &contexts.MetadataContext{}

			ctx := core.ExecutionContext{
				Data:           map[string]any{"test": "value"},
				Configuration:  tt.configuration,
				ExecutionState: stateCtx,
				Metadata:       metadataCtx,
			}

			err := ifComponent.Execute(ctx)
			assert.NoError(t, err)
			assert.True(t, stateCtx.Passed)
			assert.True(t, stateCtx.Finished)

			// Verify that the expression is stored in metadata
			assert.NotNil(t, metadataCtx.Metadata)
			metadata, ok := metadataCtx.Metadata.(map[string]any)
			assert.True(t, ok)
			assert.Equal(t, tt.configuration["expression"], metadata["expression"])

			assert.Equal(t, tt.expectedChannel, stateCtx.Channel)
			assert.Equal(t, "if.executed", stateCtx.Type)
			assert.Len(t, stateCtx.Payloads, 1)

			// Verify payload structure follows SuperPlane conventions
			payload, ok := stateCtx.Payloads[0].(map[string]any)
			assert.True(t, ok, "payload should be a map")
			assert.Equal(t, "if.executed", payload["type"])
			assert.NotEmpty(t, payload["timestamp"])
			assert.Contains(t, payload, "data")
			data, ok := payload["data"].(map[string]any)
			assert.True(t, ok, "data should be a map")
			assert.Empty(t, data, "data should be empty")
		})
	}
}

func TestIf_Execute_NodeReferenceExpression(t *testing.T) {
	ifComponent := &If{}

	stateCtx := &contexts.ExecutionStateContext{}
	metadataCtx := &contexts.MetadataContext{}

	ctx := core.ExecutionContext{
		Data: map[string]any{
			"data": map[string]any{
				"test": "value",
			},
		},
		SourceNodeID:   "upstream-node",
		Configuration:  map[string]any{"expression": "$[\"other-node\"].data.test == 'value'"},
		ExecutionState: stateCtx,
		Metadata:       metadataCtx,
		ExpressionEnv: func(expression string) (map[string]any, error) {
			return map[string]any{
				"$": map[string]any{
					"other-node": map[string]any{
						"data": map[string]any{
							"test": "value",
						},
					},
				},
			}, nil
		},
	}

	err := ifComponent.Execute(ctx)
	assert.NoError(t, err)
	assert.True(t, stateCtx.Passed)
	assert.True(t, stateCtx.Finished)
	assert.Equal(t, ChannelNameTrue, stateCtx.Channel)
}
