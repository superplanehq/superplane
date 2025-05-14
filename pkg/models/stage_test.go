package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func Test__TimeWindowCondition(t *testing.T) {
	type testCase struct {
		description   string
		start         string
		end           string
		now           string
		weekdays      []string
		expectedError string
	}

	testCases := []testCase{
		{
			description:   "in time window, but not in allowed week day",
			start:         "08:00",
			end:           "17:00",
			weekdays:      []string{"Monday"},
			now:           "2025-01-01T00:00:00Z",
			expectedError: "current day - Wednesday - is outside week days allowed - [Monday]",
		},
		{
			description:   "start before end - not in time window",
			start:         "08:00",
			end:           "17:00",
			weekdays:      []string{"Monday"},
			now:           "2025-01-06T00:00:00Z",
			expectedError: "00:00 is not in time window 08:00-17:00",
		},
		{
			description:   "start before end - in time window",
			start:         "08:00",
			end:           "17:00",
			weekdays:      []string{"Monday"},
			now:           "2025-01-06T10:00:00Z",
			expectedError: "",
		},
		{
			description:   "start after end - not in time window",
			start:         "17:00",
			end:           "08:00",
			weekdays:      []string{"Monday", "Tuesday"},
			now:           "2025-01-06T09:00:00Z",
			expectedError: "09:00 is not in time window 17:00-08:00",
		},
		{
			description:   "start after end - in window",
			start:         "17:00",
			end:           "08:00",
			weekdays:      []string{"Monday", "Tuesday"},
			now:           "2025-01-06T18:00:00Z",
			expectedError: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			now, err := time.Parse(time.RFC3339, tc.now)
			require.NoError(t, err)
			c, err := NewTimeWindowCondition(tc.start, tc.end, tc.weekdays)
			require.NoError(t, err)

			err = c.Evaluate(&now)
			if tc.expectedError != "" {
				require.ErrorContains(t, err, tc.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
