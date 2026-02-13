package newrelic

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestRunNRQLQuery_Setup_Repro(t *testing.T) {
	component := &RunNRQLQuery{}

	testCases := []struct {
		name          string
		configuration map[string]any
		expectError   bool
	}{
		{
			name: "raw string id",
			configuration: map[string]any{
				"account": "12345",
				"query":   "SELECT count(*) FROM Transaction",
				"timeout": 10,
			},
			expectError: false,
		},
		{
			name: "manual account id fallback",
			configuration: map[string]any{
				// account field is missing/nil, simulating UI issue
				"manualAccountId": "12345",
				"query":           "SELECT count(*) FROM Transaction",
				"timeout":         10,
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := core.SetupContext{
				Configuration: tc.configuration,
				Metadata:      &contexts.MetadataContext{},
			}
			err := component.Setup(ctx)
			if tc.expectError {
				require.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
