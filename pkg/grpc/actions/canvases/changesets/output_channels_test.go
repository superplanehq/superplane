package changesets

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
)

type testOutputChannelComponent struct {
	channels []core.OutputChannel
}

func (c *testOutputChannelComponent) Name() string { return "test-component" }

func (c *testOutputChannelComponent) Label() string { return "Test Component" }

func (c *testOutputChannelComponent) Description() string { return "" }

func (c *testOutputChannelComponent) Documentation() string { return "" }

func (c *testOutputChannelComponent) Icon() string { return "" }

func (c *testOutputChannelComponent) Color() string { return "" }

func (c *testOutputChannelComponent) ExampleOutput() map[string]any { return nil }

func (c *testOutputChannelComponent) OutputChannels(any) []core.OutputChannel { return c.channels }

func (c *testOutputChannelComponent) Configuration() []configuration.Field { return nil }

func (c *testOutputChannelComponent) Hooks() []core.Hook { return nil }

func (c *testOutputChannelComponent) HandleHook(core.ActionHookContext) error { return nil }

func (c *testOutputChannelComponent) Setup(core.SetupContext) error { return nil }

func (c *testOutputChannelComponent) ProcessQueueItem(core.ProcessQueueContext) (*uuid.UUID, error) {
	return nil, nil
}

func (c *testOutputChannelComponent) Execute(core.ExecutionContext) error { return nil }

func (c *testOutputChannelComponent) HandleWebhook(core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *testOutputChannelComponent) Cancel(core.ExecutionContext) error { return nil }

func (c *testOutputChannelComponent) Cleanup(core.SetupContext) error { return nil }

func TestValidateSourceNodeOutputChannel(t *testing.T) {
	reg, err := registry.NewRegistry(&crypto.NoOpEncryptor{}, registry.HTTPOptions{})
	require.NoError(t, err)

	t.Run("unresolvable source component stays soft", func(t *testing.T) {
		err := ValidateSourceNodeOutputChannel(
			reg,
			models.Node{
				ID:   "node-a",
				Type: models.NodeTypeComponent,
				Ref: models.NodeRef{
					Component: &models.ComponentRef{Name: "missing-component"},
				},
			},
			"default",
		)

		require.NoError(t, err)
	})

	t.Run("wrong channel on resolvable component returns error", func(t *testing.T) {
		reg.Actions["test-action"] = &testOutputChannelComponent{
			channels: []core.OutputChannel{
				{Name: "success"},
				{Name: "failure"},
			},
		}

		err := ValidateSourceNodeOutputChannel(
			reg,
			models.Node{
				ID:   "node-a",
				Type: models.NodeTypeComponent,
				Ref: models.NodeRef{
					Component: &models.ComponentRef{Name: "test-action"},
				},
			},
			"default",
		)

		require.Error(t, err)
		assert.Contains(t, err.Error(), `source node node-a does not have output channel "default"`)
		assert.Contains(t, err.Error(), "success")
		assert.Contains(t, err.Error(), "failure")
	})
}
