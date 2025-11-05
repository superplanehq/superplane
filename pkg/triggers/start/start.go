package manual

import (
	"net/http"

	"github.com/superplanehq/superplane/pkg/components"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/triggers"
)

func init() {
	registry.RegisterTrigger("start", &Start{})
}

type Start struct{}

func (s *Start) Name() string {
	return "start"
}

func (s *Start) Label() string {
	return "Manual Run"
}

func (s *Start) Description() string {
	return "Start a new execution chain manually"
}

func (s *Start) Icon() string {
	return "play"
}

func (s *Start) Color() string {
	return "purple"
}

func (s *Start) Configuration() []configuration.Field {
	return []configuration.Field{}
}

func (s *Start) HandleWebhook(ctx triggers.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (s *Start) Setup(ctx triggers.TriggerContext) error {
	return nil
}

func (s *Start) Actions() []components.Action {
	return []components.Action{}
}

func (s *Start) HandleAction(ctx triggers.TriggerActionContext) error {
	return nil
}
