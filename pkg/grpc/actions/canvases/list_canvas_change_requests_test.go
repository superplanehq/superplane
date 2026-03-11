package canvases

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
)

func TestResolveChangeRequestStatusFilter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                 string
		input                string
		expectedStatuses     []string
		expectedIsConflicted *bool
		expectError          bool
	}{
		{
			name:             "all",
			input:            "all",
			expectedStatuses: nil,
		},
		{
			name:             "open",
			input:            "open",
			expectedStatuses: []string{models.CanvasChangeRequestStatusOpen},
		},
		{
			name:                 "conflicted",
			input:                "conflicted",
			expectedStatuses:     nil,
			expectedIsConflicted: boolPtr(true),
		},
		{
			name:             "rejected",
			input:            "rejected",
			expectedStatuses: []string{models.CanvasChangeRequestStatusRejected},
		},
		{
			name:             "published alias",
			input:            "merged",
			expectedStatuses: []string{models.CanvasChangeRequestStatusPublished},
		},
		{
			name:        "unsupported",
			input:       "random",
			expectError: true,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			filter, err := resolveChangeRequestStatusFilter(test.input)
			if test.expectError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.expectedStatuses, filter.statuses)
			if test.expectedIsConflicted == nil {
				assert.Nil(t, filter.isConflicted)
				return
			}
			require.NotNil(t, filter.isConflicted)
			assert.Equal(t, *test.expectedIsConflicted, *filter.isConflicted)
		})
	}
}
