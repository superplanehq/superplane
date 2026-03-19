package code

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

const DefaultCode = `export default async function execute(ctx) {
  await ctx.executionState.emit("default", "application/json", [
    {
      hello: "world",
    },
  ]);
}
`

func init() {
	registry.RegisterComponent("code", &Code{})
}

type Code struct{}

type Spec struct {
	Code    string `json:"code"`
	Timeout int    `json:"timeout"`
}

func (c *Code) Name() string {
	return "code"
}

func (c *Code) Label() string {
	return "Code"
}

func (c *Code) Description() string {
	return "Execute code"
}

func (c *Code) Documentation() string {
	return ``
}

func (c *Code) Icon() string {
	return "code"
}

func (c *Code) Color() string {
	return "blue"
}

func (c *Code) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *Code) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "code",
			Label:       "Code",
			Default:     DefaultCode,
			Description: "The code to execute",
			Required:    true,
			Type:        configuration.FieldTypeText,
			TypeOptions: &configuration.TypeOptions{
				Text: &configuration.TextTypeOptions{
					Language: "typescript",
				},
			},
		},
		{
			Name:        "timeout",
			Label:       "Timeout",
			Description: "The timeout for the code execution",
			Required:    false,
			Type:        configuration.FieldTypeNumber,
			Default:     30,
		},
	}
}

func (c *Code) Execute(ctx core.ExecutionContext) error {
	spec := Spec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return err
	}

	return ctx.Runner.ExecuteCode(spec.Code, spec.Timeout)
}

func (c *Code) Actions() []core.Action {
	return []core.Action{}
}

func (c *Code) HandleAction(ctx core.ActionContext) error {
	return fmt.Errorf("code does not support actions")
}

func (c *Code) Setup(ctx core.SetupContext) error {
	return nil
}

func (c *Code) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *Code) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *Code) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *Code) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *Code) ExampleOutput() map[string]any {
	return map[string]any{}
}
