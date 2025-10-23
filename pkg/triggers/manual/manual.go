package manual

import (
	"net/http"

	"github.com/superplanehq/superplane/pkg/components"
	"github.com/superplanehq/superplane/pkg/triggers"
)

type Manual struct{}

func (m *Manual) Name() string {
	return "manual"
}

func (m *Manual) Label() string {
	return "Manual"
}

func (m *Manual) Description() string {
	return "Start a new execution chain manually"
}

func (m *Manual) Configuration() []components.ConfigurationField {
	return []components.ConfigurationField{}
}

func (m *Manual) HandleWebhook(ctx triggers.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (m *Manual) Setup(ctx triggers.TriggerContext) error {
	return nil
}

func (m *Manual) Actions() []components.Action {
	return []components.Action{}
}

func (m *Manual) HandleAction(ctx triggers.TriggerActionContext) error {
	return nil
}
