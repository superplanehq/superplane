package telemetry

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDurationSecondsHistogramBoundaries(t *testing.T) {
	require.Equal(t, []float64{
		0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10,
	}, durationSecondsHistogramBoundaries)
	assert.Contains(t, durationSecondsHistogramBoundaries, 0.5)
	assert.Contains(t, durationSecondsHistogramBoundaries, 1.0)
	assert.Equal(t, 10.0, durationSecondsHistogramBoundaries[len(durationSecondsHistogramBoundaries)-1])
}

func TestHistogramViewsAreConfigured(t *testing.T) {
	assert.NotNil(t, durationSecondsHistogramView())
	assert.NotNil(t, httpServerDurationHistogramView())
}
