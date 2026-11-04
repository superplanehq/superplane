package factory

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestUpdatePullRequestBodySectionSchedulesPublicationForCurrentRun(t *testing.T) {
	requests := &contexts.RequestContext{}
	state := &contexts.ExecutionStateContext{}
	metadata := &contexts.MetadataContext{}
	err := (&UpdatePullRequestBodySection{}).Execute(core.ExecutionContext{RunID: uuid.New(), Configuration: map[string]any{"pullRequestId": uuid.NewString(), "sectionKey": "notes", "body": "arbitrary"}, Requests: requests, Metadata: metadata, ExecutionState: state, BaseURL: "https://example.com"})
	require.NoError(t, err)
	assert.Equal(t, UpdatePullRequestBodySectionHook, requests.Action)
	assert.Equal(t, time.Second, requests.Duration)
	assert.False(t, state.Finished)
}

func TestUpdatePullRequestBodySectionRejectsMissingRunBeforeScheduling(t *testing.T) {
	requests := &contexts.RequestContext{}
	err := (&UpdatePullRequestBodySection{}).Execute(core.ExecutionContext{Requests: requests})
	require.Error(t, err)
	assert.Empty(t, requests.Action)
}

func TestUpdatePullRequestBodySectionRejectsSectionKeyExpressions(t *testing.T) {
	component := &UpdatePullRequestBodySection{}
	config := map[string]any{"pullRequestId": uuid.NewString(), "sectionKey": `{{ "release-notes" }}`, "body": "arbitrary"}

	err := configuration.ValidateConfiguration(component.Configuration(), config)
	require.ErrorContains(t, err, "expressions are not supported")

	err = component.Setup(core.SetupContext{Configuration: config})
	require.ErrorContains(t, err, "does not support expressions")

	config["sectionKey"] = "release-notes"
	require.NoError(t, configuration.ValidateConfiguration(component.Configuration(), config))
	require.NoError(t, component.Setup(core.SetupContext{Configuration: config}))
}
