package runagent

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
)

type recordingUsage struct {
	records []core.UsageRecord
}

func (r *recordingUsage) Record(record core.UsageRecord) error {
	r.records = append(r.records, record)
	return nil
}

func TestRecordSessionUsage(t *testing.T) {
	t.Parallel()

	recorder := &recordingUsage{}
	RecordSessionUsage(recorder, nil, &ManagedSession{
		Model: "claude-sonnet-4-6",
		Usage: &ManagedSessionUsage{
			InputTokens:              100,
			OutputTokens:             20,
			CacheReadInputTokens:     4,
			CacheCreationInputTokens: 6,
		},
	})
	require.Len(t, recorder.records, 1)
	assert.Equal(t, "anthropic", recorder.records[0].Provider)
	assert.Equal(t, "claude-sonnet-4-6", recorder.records[0].Model)
	assert.Equal(t, int64(130), recorder.records[0].TotalTokens)

	RecordSessionUsage(recorder, nil, &ManagedSession{Usage: &ManagedSessionUsage{}})
	assert.Len(t, recorder.records, 1)
}
