package sentry

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
)

func Test__UpdateIssue__Setup(t *testing.T) {
	component := &UpdateIssue{}

	t.Run("issue ID is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: UpdateIssueConfiguration{IssueID: ""},
		})

		require.ErrorContains(t, err, "issue ID is required")
	})

	t.Run("valid configuration", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: UpdateIssueConfiguration{IssueID: "123456789"},
		})

		require.NoError(t, err)
	})

	t.Run("invalid configuration -> decode error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: "invalid-config",
		})

		require.ErrorContains(t, err, "failed to decode configuration")
	})
}

func Test__UpdateIssue__Name(t *testing.T) {
	component := &UpdateIssue{}
	assert.Equal(t, "sentry.updateIssue", component.Name())
}

func Test__UpdateIssue__Label(t *testing.T) {
	component := &UpdateIssue{}
	assert.Equal(t, "Update Issue", component.Label())
}

func Test__UpdateIssue__Configuration(t *testing.T) {
	component := &UpdateIssue{}
	config := component.Configuration()

	assert.Len(t, config, 5)

	// Check issueId field
	assert.Equal(t, "issueId", config[0].Name)
	assert.True(t, config[0].Required)

	// Check status field
	assert.Equal(t, "status", config[1].Name)
	assert.False(t, config[1].Required)

	// Check assignedTo field
	assert.Equal(t, "assignedTo", config[2].Name)
	assert.False(t, config[2].Required)

	// Check hasSeen field
	assert.Equal(t, "hasSeen", config[3].Name)
	assert.False(t, config[3].Required)

	// Check isBookmarked field
	assert.Equal(t, "isBookmarked", config[4].Name)
	assert.False(t, config[4].Required)
}

func Test__UpdateIssue__OutputChannels(t *testing.T) {
	component := &UpdateIssue{}
	channels := component.OutputChannels(nil)

	assert.Len(t, channels, 1)
	assert.Equal(t, core.DefaultOutputChannel, channels[0])
}

func Test__UpdateIssue__Actions(t *testing.T) {
	component := &UpdateIssue{}
	actions := component.Actions()

	assert.Len(t, actions, 0)
}

func Test__UpdateIssue__HandleWebhook(t *testing.T) {
	component := &UpdateIssue{}

	code, err := component.HandleWebhook(core.WebhookRequestContext{})

	assert.Equal(t, 200, code)
	assert.NoError(t, err)
}

func Test__UpdateIssue__Cancel(t *testing.T) {
	component := &UpdateIssue{}

	err := component.Cancel(core.ExecutionContext{})

	assert.NoError(t, err)
}

func Test__UpdateIssue__Cleanup(t *testing.T) {
	component := &UpdateIssue{}

	err := component.Cleanup(core.SetupContext{})

	assert.NoError(t, err)
}
