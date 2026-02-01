package ssh

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
)

func Test__HostMetadata__Name(t *testing.T) {
	h := &HostMetadata{}
	assert.Equal(t, "ssh.hostMetadata", h.Name())
}

func Test__HostMetadata__Label(t *testing.T) {
	h := &HostMetadata{}
	assert.Equal(t, "Host Metadata", h.Label())
}

func Test__HostMetadata__Description(t *testing.T) {
	h := &HostMetadata{}
	assert.NotEmpty(t, h.Description())
}

func Test__HostMetadata__Icon(t *testing.T) {
	h := &HostMetadata{}
	assert.Equal(t, "server", h.Icon())
}

func Test__HostMetadata__Color(t *testing.T) {
	h := &HostMetadata{}
	assert.Equal(t, "green", h.Color())
}

func Test__HostMetadata__OutputChannels(t *testing.T) {
	h := &HostMetadata{}
	channels := h.OutputChannels(nil)

	// Host metadata uses default channel
	assert.Len(t, channels, 0)
}

func Test__HostMetadata__Configuration(t *testing.T) {
	h := &HostMetadata{}
	fields := h.Configuration()

	require.Len(t, fields, 1)
	assert.Equal(t, "host", fields[0].Name)
}

func Test__HostMetadata__Setup(t *testing.T) {
	h := &HostMetadata{}

	t.Run("success with valid configuration", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"host": "user@example.com:22",
			},
		}

		err := h.Setup(ctx)
		assert.NoError(t, err)
	})

	t.Run("failure with missing host", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{},
		}

		err := h.Setup(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "host is required")
	})
}

func Test__HostMetadata__ProcessQueueItem(t *testing.T) {
	h := &HostMetadata{}
	var executionID *uuid.UUID
	ctx := core.ProcessQueueContext{
		DefaultProcessing: func() (*uuid.UUID, error) {
			id := uuid.New()
			executionID = &id
			return executionID, nil
		},
	}

	result, err := h.ProcessQueueItem(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, executionID, result)
}

func Test__HostMetadata__Actions(t *testing.T) {
	h := &HostMetadata{}
	actions := h.Actions()

	assert.Len(t, actions, 0)
}

func Test__HostMetadata__HandleAction(t *testing.T) {
	h := &HostMetadata{}
	ctx := core.ActionContext{}

	err := h.HandleAction(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no actions defined")
}

func Test__HostMetadata__HandleWebhook(t *testing.T) {
	h := &HostMetadata{}
	ctx := core.WebhookRequestContext{}

	status, err := h.HandleWebhook(ctx)
	assert.Error(t, err)
	assert.Equal(t, 404, status)
}
