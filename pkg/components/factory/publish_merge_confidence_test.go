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

func TestPublishMergeConfidenceSchedulesPublicationForCurrentRun(t *testing.T) {
	requests := &contexts.RequestContext{}
	state := &contexts.ExecutionStateContext{}
	metadata := &contexts.MetadataContext{}
	err := (&PublishMergeConfidence{}).Execute(core.ExecutionContext{RunID: uuid.New(), Requests: requests, Metadata: metadata, ExecutionState: state, BaseURL: "https://example.com"})
	require.NoError(t, err)
	assert.Equal(t, PublishMergeConfidenceHook, requests.Action)
	assert.Equal(t, time.Second, requests.Duration)
	assert.False(t, state.Finished)
}

func TestPublishMergeConfidenceRejectsMissingRunBeforeScheduling(t *testing.T) {
	requests := &contexts.RequestContext{}
	err := (&PublishMergeConfidence{}).Execute(core.ExecutionContext{Requests: requests})
	require.Error(t, err)
	assert.Empty(t, requests.Action)
}
