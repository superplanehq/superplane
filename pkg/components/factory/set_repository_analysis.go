package factory

import (
	"errors"
	"net/http"

	"github.com/go-viper/mapstructure/v2"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

const SetRepositoryAnalysisComponentName = "setRepositoryAnalysis"

func init() {
	registry.RegisterAction(SetRepositoryAnalysisComponentName, &SetRepositoryAnalysis{})
}

type SetRepositoryAnalysis struct{}

type SetRepositoryAnalysisConfiguration struct {
	Document  string `json:"document" mapstructure:"document"`
	CommitSHA string `json:"commitSha" mapstructure:"commitSha"`
	Status    string `json:"status" mapstructure:"status"`
	Error     string `json:"error" mapstructure:"error"`
}

func (c *SetRepositoryAnalysis) Name() string  { return SetRepositoryAnalysisComponentName }
func (c *SetRepositoryAnalysis) Label() string { return "Set Repository Analysis" }
func (c *SetRepositoryAnalysis) Description() string {
	return "Store the static repository setup analysis"
}
func (c *SetRepositoryAnalysis) Icon() string  { return "factory" }
func (c *SetRepositoryAnalysis) Color() string { return "blue" }

func (c *SetRepositoryAnalysis) Documentation() string {
	return "Stores a versioned Suss repository analysis on the workspace. This component can only be used in factory-owned apps."
}

func (c *SetRepositoryAnalysis) ExampleOutput() map[string]any {
	return map[string]any{
		"timestamp": "2026-01-01T00:00:00Z",
		"type":      "repository.analysisStored",
		"data":      map[string]any{"status": "ready"},
	}
}

func (c *SetRepositoryAnalysis) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *SetRepositoryAnalysis) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "document",
			Label:       "Suss document",
			Description: "Versioned Suss JSON document",
			Type:        configuration.FieldTypeText,
			Required:    false,
		},
		{
			Name:        "commitSha",
			Label:       "Commit SHA",
			Description: "Repository commit that Suss analyzed",
			Type:        configuration.FieldTypeString,
			Required:    false,
		},
		{
			Name:        "status",
			Label:       "Status",
			Description: "Repository analysis status",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Default:     "ready",
		},
		{
			Name:        "error",
			Label:       "Error",
			Description: "Failure reason",
			Type:        configuration.FieldTypeString,
			Required:    false,
		},
	}
}

func (c *SetRepositoryAnalysis) ValidateNodeConfiguration(config map[string]any) error {
	decoded := SetRepositoryAnalysisConfiguration{}
	if err := mapstructure.Decode(config, &decoded); err != nil {
		return err
	}
	switch decoded.Status {
	case "running", "failed":
		return nil
	case "ready":
		if decoded.Document == "" || decoded.CommitSHA == "" {
			return errors.New("document and commit SHA are required for a ready analysis")
		}
		return nil
	default:
		return errors.New("status must be running, ready, or failed")
	}
}

func (c *SetRepositoryAnalysis) Execute(ctx core.ExecutionContext) error {
	config := SetRepositoryAnalysisConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return err
	}
	if err := ctx.Factory.SetRepositoryAnalysis(core.RepositoryAnalysisParams{
		Document:  config.Document,
		CommitSHA: config.CommitSHA,
		Status:    config.Status,
		Error:     config.Error,
	}); err != nil {
		return err
	}
	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"repository.analysisStored",
		[]any{map[string]any{"status": config.Status}},
	)
}

func (c *SetRepositoryAnalysis) Setup(ctx core.SetupContext) error      { return nil }
func (c *SetRepositoryAnalysis) Cancel(ctx core.ExecutionContext) error { return nil }
func (c *SetRepositoryAnalysis) Cleanup(ctx core.SetupContext) error    { return nil }
func (c *SetRepositoryAnalysis) Hooks() []core.Hook                     { return nil }
func (c *SetRepositoryAnalysis) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
func (c *SetRepositoryAnalysis) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}
