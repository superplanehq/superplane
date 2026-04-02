package grafana

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func Test__validateAnnotationTimeRangeMS__OK(t *testing.T) {
	require.NoError(t, validateAnnotationTimeRangeMS(100, 200))
	require.NoError(t, validateAnnotationTimeRangeMS(100, 100))
	require.NoError(t, validateAnnotationTimeRangeMS(0, 200))
	require.NoError(t, validateAnnotationTimeRangeMS(100, 0))
}

func Test__validateAnnotationTimeRangeMS__RejectsInvertedRange(t *testing.T) {
	err := validateAnnotationTimeRangeMS(200, 100)
	require.Error(t, err)
	require.Contains(t, err.Error(), "timeEnd must be at or after time")
}

func Test__parseAnnotationTime__ParsesRFC3339(t *testing.T) {
	tm, err := parseAnnotationTime("2024-06-01T12:00:00Z")
	require.NoError(t, err)
	require.Equal(t, 12, tm.UTC().Hour())
}

func Test__parseAnnotationTime__ParsesLocalWallTime(t *testing.T) {
	tm, err := parseAnnotationTime("2024-06-01T15:04")
	require.NoError(t, err)
	require.Equal(t, 15, tm.Hour())
	require.Equal(t, 4, tm.Minute())
	require.Equal(t, time.Local, tm.Location())
}
