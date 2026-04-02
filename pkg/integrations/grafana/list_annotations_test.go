package grafana

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test__validateListAnnotationTimeRangeMS__OK(t *testing.T) {
	require.NoError(t, validateListAnnotationTimeRangeMS(100, 200))
	require.NoError(t, validateListAnnotationTimeRangeMS(100, 100))
	require.NoError(t, validateListAnnotationTimeRangeMS(0, 200))
	require.NoError(t, validateListAnnotationTimeRangeMS(100, 0))
}

func Test__validateListAnnotationTimeRangeMS__RejectsInvertedRange(t *testing.T) {
	err := validateListAnnotationTimeRangeMS(200, 100)
	require.Error(t, err)
	require.Contains(t, err.Error(), "to must be at or after from")
}
