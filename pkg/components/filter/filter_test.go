package filter

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestFilter_Execute_EmitsEmptyEvents(t *testing.T) {
	tests := []struct {
		name                 string
		configuration        map[string]any
		inputData            any
		expectedOutputsCount int
		expectedChannel      string
	}{
		{
			name:                 "filter with true condition emits empty event",
			configuration:        map[string]any{"expression": "true"},
			inputData:            map[string]any{"test": "value"},
			expectedOutputsCount: 1,
			expectedChannel:      "default",
		},
		{
			name:                 "filter with false condition emits empty event",
			configuration:        map[string]any{"expression": "false"},
			inputData:            map[string]any{"test": "value"},
			expectedOutputsCount: 0,
			expectedChannel:      "",
		},
		{
			name:                 "filter with complex true condition emits empty event",
			configuration:        map[string]any{"expression": "$.test == 'value'"},
			inputData:            map[string]any{"test": "value"},
			expectedOutputsCount: 1,
			expectedChannel:      "default",
		},
		{
			name:                 "filter with complex false condition emits empty event",
			configuration:        map[string]any{"expression": "$.test == 'different'"},
			inputData:            map[string]any{"test": "value"},
			expectedOutputsCount: 0,
			expectedChannel:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			filter := &Filter{}

			stateCtx := &contexts.ExecutionStateContext{}
			metadataCtx := &contexts.MetadataContext{}

			ctx := core.ExecutionContext{
				Data:           tt.inputData,
				Configuration:  tt.configuration,
				ExecutionState: stateCtx,
				Metadata:       metadataCtx,
			}

			err := filter.Execute(ctx)

			assert.NoError(t, err)
			assert.True(t, stateCtx.Passed)
			assert.True(t, stateCtx.Finished)

			// Verify that the expression is stored in metadata
			assert.NotNil(t, metadataCtx.Metadata)
			metadata, ok := metadataCtx.Metadata.(map[string]any)
			assert.True(t, ok)
			assert.Equal(t, tt.configuration["expression"], metadata["expression"])

			if tt.expectedOutputsCount > 0 {
				assert.Equal(t, tt.expectedChannel, stateCtx.Channel)
				assert.Equal(t, "filter.executed", stateCtx.Type)
				assert.Len(t, stateCtx.Payloads, 1)

				// Verify payload structure follows SuperPlane conventions
				payload, ok := stateCtx.Payloads[0].(map[string]any)
				assert.True(t, ok, "payload should be a map")
				assert.Equal(t, "filter.executed", payload["type"])
				assert.NotEmpty(t, payload["timestamp"])
				assert.Contains(t, payload, "data")
				data, ok := payload["data"].(map[string]any)
				assert.True(t, ok, "data should be a map")
				assert.Empty(t, data, "data should be empty")
			} else {
				assert.Empty(t, stateCtx.Channel)
				assert.Empty(t, stateCtx.Type)
				assert.Empty(t, stateCtx.Payloads)
			}
		})
	}
}

func TestFilter_Execute_InvalidExpression_ShouldReturnError(t *testing.T) {
	filter := &Filter{}

	stateCtx := &contexts.ExecutionStateContext{}
	metadataCtx := &contexts.MetadataContext{}

	ctx := core.ExecutionContext{
		Data:           map[string]any{"test": "value"},
		Configuration:  map[string]any{"expression": "invalid expression syntax +++"},
		ExecutionState: stateCtx,
		Metadata:       metadataCtx,
	}

	err := filter.Execute(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expression compilation failed")
}

func TestFilter_Execute_NonBooleanResult_ShouldReturnError(t *testing.T) {
	filter := &Filter{}

	stateCtx := &contexts.ExecutionStateContext{}
	metadataCtx := &contexts.MetadataContext{}

	ctx := core.ExecutionContext{
		Data:           map[string]any{"test": "value"},
		Configuration:  map[string]any{"expression": "$.test"},
		ExecutionState: stateCtx,
		Metadata:       metadataCtx,
	}

	err := filter.Execute(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expression must evaluate to boolean")
}

func TestFilter_Execute_NodeReferenceExpression(t *testing.T) {
	filter := &Filter{}

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

	err := filter.Execute(ctx)
	assert.NoError(t, err)
	assert.True(t, stateCtx.Passed)
	assert.True(t, stateCtx.Finished)
	assert.Equal(t, core.DefaultOutputChannel.Name, stateCtx.Channel)
}
