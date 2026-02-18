package jsruntime

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

// JSComponentAdapter wraps a JavaScript component file and implements core.Component.
// It delegates Execute() and Setup() to the goja runtime.
type JSComponentAdapter struct {
	runtime        *Runtime
	source         string
	name           string
	label          string
	description    string
	documentation  string
	icon           string
	color          string
	config         []configuration.Field
	outputChannels []core.OutputChannel
}

func NewJSComponentAdapter(rt *Runtime, source string, def *ComponentDefinition, registryName string) (*JSComponentAdapter, error) {
	config, err := def.ParseConfiguration()
	if err != nil {
		return nil, fmt.Errorf("failed to parse configuration: %w", err)
	}

	channels, err := def.ParseOutputChannels()
	if err != nil {
		return nil, fmt.Errorf("failed to parse output channels: %w", err)
	}

	name := registryName
	if def.Name != "" {
		name = def.Name
	}

	label := def.Label
	if label == "" {
		label = registryName
	}

	return &JSComponentAdapter{
		runtime:        rt,
		source:         source,
		name:           name,
		label:          label,
		description:    def.Description,
		documentation:  def.Documentation,
		icon:           def.Icon,
		color:          def.Color,
		config:         config,
		outputChannels: channels,
	}, nil
}

func (a *JSComponentAdapter) Name() string          { return a.name }
func (a *JSComponentAdapter) Label() string         { return a.label }
func (a *JSComponentAdapter) Description() string   { return a.description }
func (a *JSComponentAdapter) Documentation() string { return a.documentation }
func (a *JSComponentAdapter) Icon() string          { return a.icon }
func (a *JSComponentAdapter) Color() string         { return a.color }

func (a *JSComponentAdapter) ExampleOutput() map[string]any {
	return map[string]any{}
}

func (a *JSComponentAdapter) OutputChannels(_ any) []core.OutputChannel {
	return a.outputChannels
}

func (a *JSComponentAdapter) Configuration() []configuration.Field {
	if a.config == nil {
		return []configuration.Field{}
	}

	return a.config
}

func (a *JSComponentAdapter) Setup(ctx core.SetupContext) error {
	return a.runtime.Setup(a.source, ctx)
}

func (a *JSComponentAdapter) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (a *JSComponentAdapter) Execute(ctx core.ExecutionContext) error {
	return a.runtime.Execute(a.source, ctx)
}

func (a *JSComponentAdapter) Actions() []core.Action {
	return []core.Action{}
}

func (a *JSComponentAdapter) HandleAction(_ core.ActionContext) error {
	return fmt.Errorf("JavaScript components do not support actions")
}

func (a *JSComponentAdapter) HandleWebhook(_ core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (a *JSComponentAdapter) Cancel(_ core.ExecutionContext) error {
	return nil
}

func (a *JSComponentAdapter) Cleanup(_ core.SetupContext) error {
	return nil
}
