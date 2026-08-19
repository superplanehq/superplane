package runner

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBillableSeconds(t *testing.T) {
	for _, tc := range []struct {
		name     string
		duration time.Duration
		expected int64
	}{
		{"sub-second rounds up", 117 * time.Millisecond, 1},
		{"whole second", time.Second, 1},
		{"partial second rounds up", 7739 * time.Millisecond, 8},
		{"exact multiple", 5 * time.Minute, 300},
		{"zero", 0, 0},
		{"sub-second negative from clock skew", -117 * time.Millisecond, 0},
		{"negative", -30 * time.Second, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, billableSeconds(tc.duration))
		})
	}
}
