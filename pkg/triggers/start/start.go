package manual

import (
	"net/http"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
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

func (s *Start) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (s *Start) Setup(ctx core.TriggerContext) error {
	return nil
}

func (s *Start) Actions() []core.Action {
	return []core.Action{}
}

func (s *Start) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}
