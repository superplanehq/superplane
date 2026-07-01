package runner

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReadLiveLogRecordsStopsAfterLimitEvenWhenNextLineIsInvalid(t *testing.T) {
	result, err := readLiveLogRecords(strings.NewReader(`{"type":"line","text":"first"}`+"\nnot-json\n"), 1)

	require.NoError(t, err)
	require.Len(t, result.Records, 1)
	require.Equal(t, "first", result.Records[0].Text)
	require.True(t, result.Truncated)
}
