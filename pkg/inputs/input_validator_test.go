package inputs

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/protos/superplane"
)

func Test__InputValidator(t *testing.T) {
	type testCase struct {
		name          string
		inputs        []*superplane.InputDefinition
		inputMappings []*superplane.InputMapping
		outputs       []*superplane.OutputDefinition
		connections   []*superplane.Connection
		expectErr     bool
		errMessage    string
	}

	testCases := []testCase{
		{
			name:          "nothing is defined",
			inputs:        []*superplane.InputDefinition{},
			outputs:       []*superplane.OutputDefinition{},
			inputMappings: []*superplane.InputMapping{},
			connections:   []*superplane.Connection{},
			expectErr:     false,
		},
		{
			name:       "input name is empty",
			inputs:     []*superplane.InputDefinition{{Name: ""}},
			expectErr:  true,
			errMessage: "empty input name",
		},
		{
			name:       "output name is empty",
			outputs:    []*superplane.OutputDefinition{{Name: ""}},
			expectErr:  true,
			errMessage: "empty output name",
		},
		{
			name:       "input name is defined multiple times",
			inputs:     []*superplane.InputDefinition{{Name: "a"}, {Name: "a"}},
			expectErr:  true,
			errMessage: "input a defined multiple times",
		},
		{
			name:       "output name is defined multiple times",
			outputs:    []*superplane.OutputDefinition{{Name: "a"}, {Name: "a"}},
			expectErr:  true,
			errMessage: "output a defined multiple times",
		},
		{
			name:   "invalid mapping - no value definitions",
			inputs: []*superplane.InputDefinition{{Name: "a"}},
			inputMappings: []*superplane.InputMapping{
				{
					Values: []*superplane.ValueDefinition{},
				},
			},
			connections: []*superplane.Connection{},
			expectErr:   true,
			errMessage:  "invalid mapping [0]: no value definitions",
		},
		{
			name:   "invalid mapping - value is not defined",
			inputs: []*superplane.InputDefinition{{Name: "a"}},
			inputMappings: []*superplane.InputMapping{
				{
					Values: []*superplane.ValueDefinition{
						{Name: "b"},
					},
				},
			},
			connections: []*superplane.Connection{},
			expectErr:   true,
			errMessage:  "invalid mapping [0]: value is not defined",
		},
		{
			name:   "invalid mapping - value and valueFrom used",
			inputs: []*superplane.InputDefinition{{Name: "a"}},
			inputMappings: []*superplane.InputMapping{
				{
					Values: []*superplane.ValueDefinition{
						{Name: "b", Value: "c", ValueFrom: &superplane.ValueFrom{}},
					},
				},
			},
			connections: []*superplane.Connection{},
			expectErr:   true,
			errMessage:  "invalid mapping [0]: cannot use value and valueFrom at the same time",
		},
		{
			name:   "invalid mapping - invalid valueFrom - no source defined",
			inputs: []*superplane.InputDefinition{{Name: "a"}},
			inputMappings: []*superplane.InputMapping{
				{
					Values: []*superplane.ValueDefinition{
						{Name: "b", ValueFrom: &superplane.ValueFrom{}},
					},
				},
			},
			connections: []*superplane.Connection{},
			expectErr:   true,
			errMessage:  "invalid mapping [0]: invalid valueFrom for b: no source defined",
		},
		{
			name:   "invalid mapping - invalid valueFrom - multiple sources",
			inputs: []*superplane.InputDefinition{{Name: "a"}},
			inputMappings: []*superplane.InputMapping{
				{
					Values: []*superplane.ValueDefinition{
						{Name: "b", ValueFrom: &superplane.ValueFrom{
							EventData:     &superplane.ValueFromEventData{},
							LastExecution: &superplane.ValueFromLastExecution{},
						}},
					},
				},
			},
			connections: []*superplane.Connection{},
			expectErr:   true,
			errMessage:  "invalid mapping [0]: invalid valueFrom for b: cannot use multiple sources at the same time",
		},
		{
			name:   "invalid mapping - invalid valueFrom - invalid event data - empty connection",
			inputs: []*superplane.InputDefinition{{Name: "a"}},
			inputMappings: []*superplane.InputMapping{
				{
					Values: []*superplane.ValueDefinition{
						{Name: "b", ValueFrom: &superplane.ValueFrom{
							EventData: &superplane.ValueFromEventData{Connection: ""},
						}},
					},
				},
			},
			connections: []*superplane.Connection{},
			expectErr:   true,
			errMessage:  "invalid mapping [0]: invalid valueFrom for b: invalid event data: empty connection",
		},
		{
			name:   "invalid mapping - invalid valueFrom - invalid event data - empty expression",
			inputs: []*superplane.InputDefinition{{Name: "a"}},
			inputMappings: []*superplane.InputMapping{
				{
					Values: []*superplane.ValueDefinition{
						{Name: "b", ValueFrom: &superplane.ValueFrom{
							EventData: &superplane.ValueFromEventData{Connection: "ok", Expression: ""},
						}},
					},
				},
			},
			connections: []*superplane.Connection{},
			expectErr:   true,
			errMessage:  "invalid mapping [0]: invalid valueFrom for b: invalid event data: empty expression",
		},
		{
			name:   "invalid mapping - invalid valueFrom - invalid last execution - empty results",
			inputs: []*superplane.InputDefinition{{Name: "a"}},
			inputMappings: []*superplane.InputMapping{
				{
					Values: []*superplane.ValueDefinition{
						{Name: "b", ValueFrom: &superplane.ValueFrom{
							LastExecution: &superplane.ValueFromLastExecution{},
						}},
					},
				},
			},
			connections: []*superplane.Connection{},
			expectErr:   true,
			errMessage:  "invalid mapping [0]: invalid valueFrom for b: invalid last execution: empty results",
		},
		{
			name:   "invalid mapping - invalid valueFrom - invalid last execution - invalid execution result",
			inputs: []*superplane.InputDefinition{{Name: "a"}},
			inputMappings: []*superplane.InputMapping{
				{
					Values: []*superplane.ValueDefinition{
						{Name: "b", ValueFrom: &superplane.ValueFrom{
							LastExecution: &superplane.ValueFromLastExecution{
								Results: []superplane.Execution_Result{superplane.Execution_RESULT_UNKNOWN},
							},
						}},
					},
				},
			},
			connections: []*superplane.Connection{},
			expectErr:   true,
			errMessage:  "invalid mapping [0]: invalid valueFrom for b: invalid last execution: invalid execution result RESULT_UNKNOWN",
		},
		{
			name:   "invalid mapping - invalid when - no triggered by",
			inputs: []*superplane.InputDefinition{{Name: "a"}},
			inputMappings: []*superplane.InputMapping{
				{
					Values: []*superplane.ValueDefinition{
						{Name: "b"},
					},
					When: &superplane.InputMapping_When{},
				},
			},
			connections: []*superplane.Connection{},
			expectErr:   true,
			errMessage:  "invalid mapping [0]: missing triggered by",
		},
		{
			name:   "invalid mapping - invalid when - no connection name",
			inputs: []*superplane.InputDefinition{{Name: "a"}},
			inputMappings: []*superplane.InputMapping{
				{
					Values: []*superplane.ValueDefinition{
						{Name: "a"},
					},
					When: &superplane.InputMapping_When{
						TriggeredBy: &superplane.InputMapping_WhenTriggeredBy{
							Connection: "",
						},
					},
				},
			},
			connections: []*superplane.Connection{},
			expectErr:   true,
			errMessage:  "invalid mapping [0]: empty connection name",
		},
		{
			name:   "no duplicate mapping for connection",
			inputs: []*superplane.InputDefinition{{Name: "a"}},
			inputMappings: []*superplane.InputMapping{
				{
					Values: []*superplane.ValueDefinition{
						{Name: "a", Value: "a"},
					},
					When: &superplane.InputMapping_When{
						TriggeredBy: &superplane.InputMapping_WhenTriggeredBy{
							Connection: "a",
						},
					},
				},
				{
					Values: []*superplane.ValueDefinition{
						{Name: "a", Value: "b"},
					},
					When: &superplane.InputMapping_When{
						TriggeredBy: &superplane.InputMapping_WhenTriggeredBy{
							Connection: "a",
						},
					},
				},
			},
			connections: []*superplane.Connection{
				{Name: "a"},
			},
			expectErr:  true,
			errMessage: "multiple input mappings for connection a",
		},
		{
			name:   "undefined input",
			inputs: []*superplane.InputDefinition{{Name: "a"}, {Name: "b"}},
			inputMappings: []*superplane.InputMapping{
				{
					Values: []*superplane.ValueDefinition{
						{Name: "a", Value: "a"},
					},
				},
			},
			connections: []*superplane.Connection{},
			expectErr:   true,
			errMessage:  "mapping [0]: input b not defined",
		},
		{
			name:   "mapping defines input that does not exist",
			inputs: []*superplane.InputDefinition{{Name: "a"}, {Name: "b"}},
			inputMappings: []*superplane.InputMapping{
				{
					Values: []*superplane.ValueDefinition{
						{Name: "a", Value: "a"},
						{Name: "b", Value: "b"},
						{Name: "c", Value: "c"},
					},
				},
			},
			connections: []*superplane.Connection{},
			expectErr:   true,
			errMessage:  "mapping [0]: value definition [2]: input c not defined",
		},
		{
			name:   "value definition for same input define twice",
			inputs: []*superplane.InputDefinition{{Name: "a"}, {Name: "b"}},
			inputMappings: []*superplane.InputMapping{
				{
					Values: []*superplane.ValueDefinition{
						{Name: "a", Value: "a"},
						{Name: "b", Value: "b"},
						{Name: "b", Value: "b"},
					},
				},
			},
			connections: []*superplane.Connection{},
			expectErr:   true,
			errMessage:  "mapping [0]: input b defined multiple times",
		},
		{
			name:   "connection that does not exist in triggered by",
			inputs: []*superplane.InputDefinition{{Name: "a"}, {Name: "b"}},
			inputMappings: []*superplane.InputMapping{
				{
					When: &superplane.InputMapping_When{
						TriggeredBy: &superplane.InputMapping_WhenTriggeredBy{
							Connection: "source-2",
						},
					},
					Values: []*superplane.ValueDefinition{
						{Name: "a", Value: "a"},
						{Name: "b", Value: "b"},
					},
				},
			},
			connections: []*superplane.Connection{
				{Name: "source-1"},
			},
			expectErr:  true,
			errMessage: "mapping [0]: connection source-2 does not exist",
		},
		{
			name:   "connection that does not exist in valueFrom.eventData",
			inputs: []*superplane.InputDefinition{{Name: "a"}, {Name: "b"}},
			inputMappings: []*superplane.InputMapping{
				{
					Values: []*superplane.ValueDefinition{
						{Name: "a", Value: "a"},
						{Name: "b", ValueFrom: &superplane.ValueFrom{
							EventData: &superplane.ValueFromEventData{
								Connection: "source-2",
								Expression: "ref",
							},
						}},
					},
				},
			},
			connections: []*superplane.Connection{
				{Name: "source-1"},
			},
			expectErr:  true,
			errMessage: "mapping [0]: value definition [1]: connection source-2 does not exist",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			validator := NewValidator(
				WithInputs(testCase.inputs),
				WithOutputs(testCase.outputs),
				WithInputMappings(testCase.inputMappings),
				WithConnections(testCase.connections),
			)

			err := validator.Validate()
			if testCase.expectErr {
				assert.ErrorContains(t, err, testCase.errMessage)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
