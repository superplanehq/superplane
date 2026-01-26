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

func (s *Start) Documentation() string {
	return `The Manual Run trigger allows you to start workflow executions manually from the SuperPlane UI.

## Use Cases

- **Testing workflows**: Manually trigger workflows during development and testing
- **One-off tasks**: Run workflows on-demand for specific operations
- **Debugging**: Manually execute workflows to debug issues
- **Ad-hoc processing**: Process data when needed without automation

## How It Works

1. Add the Manual Run trigger as the starting node of your workflow
2. Click the "Run" button in the workflow UI to start an execution
3. The workflow begins immediately with empty event data

## Configuration

The Manual Run trigger requires no configuration. It's ready to use immediately after being added to a workflow.

## Event Data

Manual runs start with an empty event payload. You can use this as a starting point and add data through subsequent components.`
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
