package ssh

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
)

func Test__ExecuteScript__Name(t *testing.T) {
	e := &ExecuteScript{}
	assert.Equal(t, "ssh.executeScript", e.Name())
}

func Test__ExecuteScript__Label(t *testing.T) {
	e := &ExecuteScript{}
	assert.Equal(t, "Execute Script", e.Label())
}

func Test__ExecuteScript__Description(t *testing.T) {
	e := &ExecuteScript{}
	assert.NotEmpty(t, e.Description())
}

func Test__ExecuteScript__Icon(t *testing.T) {
	e := &ExecuteScript{}
	assert.Equal(t, "file-code", e.Icon())
}

func Test__ExecuteScript__Color(t *testing.T) {
	e := &ExecuteScript{}
	assert.Equal(t, "purple", e.Color())
}

func Test__ExecuteScript__OutputChannels(t *testing.T) {
	e := &ExecuteScript{}
	channels := e.OutputChannels(nil)

	require.Len(t, channels, 2)
	assert.Equal(t, ExecuteScriptSuccessChannel, channels[0].Name)
	assert.Equal(t, ExecuteScriptFailedChannel, channels[1].Name)
}

func Test__ExecuteScript__Configuration(t *testing.T) {
	e := &ExecuteScript{}
	fields := e.Configuration()

	require.Len(t, fields, 5)
	assert.Equal(t, "host", fields[0].Name)
	assert.Equal(t, "script", fields[1].Name)
	assert.Equal(t, "interpreter", fields[2].Name)
	assert.Equal(t, "workingDirectory", fields[3].Name)
	assert.Equal(t, "timeout", fields[4].Name)
}

func Test__ExecuteScript__Setup(t *testing.T) {
	e := &ExecuteScript{}

	t.Run("success with valid configuration", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"host":   "user@example.com:22",
				"script": "echo 'Hello, World!'",
			},
		}

		err := e.Setup(ctx)
		assert.NoError(t, err)
	})

	t.Run("failure with missing host", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"script": "echo 'Hello, World!'",
			},
		}

		err := e.Setup(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "host is required")
	})

	t.Run("failure with missing script", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"host": "user@example.com:22",
			},
		}

		err := e.Setup(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "script is required")
	})
}

func Test__ExecuteScript__ProcessQueueItem(t *testing.T) {
	e := &ExecuteScript{}
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

func Test__ExecuteScript__Actions(t *testing.T) {
	e := &ExecuteScript{}
	actions := e.Actions()

	assert.Len(t, actions, 0)
}

func Test__ExecuteScript__HandleAction(t *testing.T) {
	e := &ExecuteScript{}
	ctx := core.ActionContext{}

	err := e.HandleAction(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no actions defined")
}

func Test__ExecuteScript__HandleWebhook(t *testing.T) {
	e := &ExecuteScript{}
	ctx := core.WebhookRequestContext{}

	status, err := e.HandleWebhook(ctx)
	assert.Error(t, err)
	assert.Equal(t, 404, status)
}
