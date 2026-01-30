package ssh

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
)

func Test__ExecuteCommand__Name(t *testing.T) {
	e := &ExecuteCommand{}
	assert.Equal(t, "ssh.executeCommand", e.Name())
}

func Test__ExecuteCommand__Label(t *testing.T) {
	e := &ExecuteCommand{}
	assert.Equal(t, "Execute Command", e.Label())
}

func Test__ExecuteCommand__Description(t *testing.T) {
	e := &ExecuteCommand{}
	assert.NotEmpty(t, e.Description())
}

func Test__ExecuteCommand__Icon(t *testing.T) {
	e := &ExecuteCommand{}
	assert.Equal(t, "terminal", e.Icon())
}

func Test__ExecuteCommand__Color(t *testing.T) {
	e := &ExecuteCommand{}
	assert.Equal(t, "blue", e.Color())
}

func Test__ExecuteCommand__OutputChannels(t *testing.T) {
	e := &ExecuteCommand{}
	channels := e.OutputChannels(nil)

	require.Len(t, channels, 2)
	assert.Equal(t, ExecuteCommandSuccessChannel, channels[0].Name)
	assert.Equal(t, ExecuteCommandFailedChannel, channels[1].Name)
}

func Test__ExecuteCommand__Configuration(t *testing.T) {
	e := &ExecuteCommand{}
	fields := e.Configuration()

	require.Len(t, fields, 4)
	assert.Equal(t, "host", fields[0].Name)
	assert.Equal(t, "command", fields[1].Name)
	assert.Equal(t, "workingDirectory", fields[2].Name)
	assert.Equal(t, "timeout", fields[3].Name)
}

func Test__ExecuteCommand__Setup(t *testing.T) {
	e := &ExecuteCommand{}

	t.Run("success with valid configuration", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"host":    "user@example.com:22",
				"command": "ls -la",
			},
		}

		err := e.Setup(ctx)
		assert.NoError(t, err)
	})

	t.Run("failure with missing host", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"command": "ls -la",
			},
		}

		err := e.Setup(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "host is required")
	})

	t.Run("failure with missing command", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"host": "user@example.com:22",
			},
		}

		err := e.Setup(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "command is required")
	})
}

func Test__ExecuteCommand__ProcessQueueItem(t *testing.T) {
	e := &ExecuteCommand{}
	var executionID *uuid.UUID
	ctx := core.ProcessQueueContext{
		DefaultProcessing: func() (*uuid.UUID, error) {
			id := uuid.New()
			executionID = &id
			return executionID, nil
		},
	}

	result, err := e.ProcessQueueItem(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, executionID, result)
}

func Test__ExecuteCommand__Actions(t *testing.T) {
	e := &ExecuteCommand{}
	actions := e.Actions()

	assert.Len(t, actions, 0)
}

func Test__ExecuteCommand__HandleAction(t *testing.T) {
	e := &ExecuteCommand{}
	ctx := core.ActionContext{}

	err := e.HandleAction(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no actions defined")
}

func Test__ExecuteCommand__HandleWebhook(t *testing.T) {
	e := &ExecuteCommand{}
	ctx := core.WebhookRequestContext{}

	status, err := e.HandleWebhook(ctx)
	assert.Error(t, err)
	assert.Equal(t, 404, status)
}
