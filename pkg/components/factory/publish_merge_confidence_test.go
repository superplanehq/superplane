package factory

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestPublishMergeConfidenceSchedulesIndependentPublication(t *testing.T) {
	requests := &contexts.RequestContext{}
	state := &contexts.ExecutionStateContext{}
	metadata := &contexts.MetadataContext{}
	err := (&PublishMergeConfidence{}).Execute(core.ExecutionContext{Configuration: map[string]any{"sourceRunId": uuid.NewString()}, Requests: requests, Metadata: metadata, ExecutionState: state, BaseURL: "https://example.com"})
	require.NoError(t, err)
	assert.Equal(t, PublishMergeConfidenceHook, requests.Action)
	assert.Equal(t, time.Second, requests.Duration)
	assert.False(t, state.Finished)
}

func TestPublishMergeConfidenceRejectsInvalidSourceBeforeScheduling(t *testing.T) {
	requests := &contexts.RequestContext{}
	err := (&PublishMergeConfidence{}).Execute(core.ExecutionContext{Configuration: map[string]any{"sourceRunId": "invalid"}, Requests: requests})
	require.Error(t, err)
	assert.Empty(t, requests.Action)
}
