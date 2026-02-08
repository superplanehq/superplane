package circleci

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type TriggerPipeline struct{}

type TriggerPipelineConfiguration struct {
	ProjectSlug string      `json:"projectSlug" mapstructure:"projectSlug"`
	Branch      string      `json:"branch" mapstructure:"branch"`
	Tag         string      `json:"tag" mapstructure:"tag"`
	Parameters  []Parameter `json:"parameters" mapstructure:"parameters"`
}

type TriggerPipelineMetadata struct {
	Project *Project `json:"project" mapstructure:"project"`
}

type Parameter struct {
	Name  string `json:"name" mapstructure:"name"`
	Value string `json:"value" mapstructure:"value"`
}

func (t *TriggerPipeline) Name() string        { return "circleci.triggerPipeline" }
func (t *TriggerPipeline) Label() string       { return "Trigger Pipeline" }
func (t *TriggerPipeline) Description() string { return "Trigger a CircleCI pipeline for a project" }

func (t *TriggerPipeline) Documentation() string {
	return `The Trigger Pipeline component triggers a CircleCI pipeline for a given project.

## Configuration

- **Project Slug**: ` + "`" + `vcs-slug/org-name/repo-name` + "`" + ` (example: ` + "`" + `gh/my-org/my-repo` + "`" + `)
- **Branch**: Branch name (mutually exclusive with Tag)
- **Tag**: Tag name (mutually exclusive with Branch)
- **Parameters**: Optional pipeline parameters (values are parsed as bool/int/string)

## Output

Emits a ` + "`" + `circleci.pipeline.triggered` + "`" + ` event containing the created pipeline ID/number/state.
`
}

func (t *TriggerPipeline) Icon() string  { return "workflow" }
func (t *TriggerPipeline) Color() string { return "gray" }

func (t *TriggerPipeline) ExampleOutput() map[string]any {
	return map[string]any{
		"id":         "5034460f-c7c4-4c43-9457-de07e2029e7b",
		"number":     25,
		"state":      "created",
		"created_at": "2025-01-01T00:00:00Z",
	}
}

func (t *TriggerPipeline) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (t *TriggerPipeline) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "projectSlug",
			Label:       "Project Slug",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "e.g. gh/my-org/my-repo",
			Description: "CircleCI project slug in the form vcs-slug/org-name/repo-name",
		},
		{
			Name:        "branch",
			Label:       "Branch",
			Type:        configuration.FieldTypeString,
			Placeholder: "e.g. main",
			Description: "Branch to run (mutually exclusive with Tag)",
		},
		{
			Name:        "tag",
			Label:       "Tag",
			Type:        configuration.FieldTypeString,
			Placeholder: "e.g. v1.2.3",
			Description: "Tag to run (mutually exclusive with Branch)",
		},
		{
			Name:  "parameters",
			Label: "Parameters",
			Type:  configuration.FieldTypeList,
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Parameter",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{Name: "name", Label: "Name", Type: configuration.FieldTypeString, Required: true},
							{Name: "value", Label: "Value", Type: configuration.FieldTypeString, Required: true},
						},
					},
				},
			},
		},
	}
}

func (t *TriggerPipeline) Setup(ctx core.SetupContext) error {
	cfg := TriggerPipelineConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &cfg); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}
	if cfg.ProjectSlug == "" {
		return errors.New("projectSlug is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	project, err := client.GetProjectBySlug(cfg.ProjectSlug)
	if err != nil {
		return fmt.Errorf("failed to fetch project: %w", err)
	}

	return ctx.Metadata.Set(TriggerPipelineMetadata{Project: project})
}

func (t *TriggerPipeline) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func parseParameterValue(v string) any {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	if strings.EqualFold(v, "true") {
		return true
	}
	if strings.EqualFold(v, "false") {
		return false
	}
	if i, err := strconv.ParseInt(v, 10, 64); err == nil {
		return i
	}
	return v
}

func (t *TriggerPipeline) Execute(ctx core.ExecutionContext) error {
	cfg := TriggerPipelineConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &cfg); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}
	if cfg.ProjectSlug == "" {
		return errors.New("projectSlug is required")
	}
	if cfg.Branch != "" && cfg.Tag != "" {
		return errors.New("branch and tag are mutually exclusive")
	}

	params := map[string]any{}
	for _, p := range cfg.Parameters {
		if strings.TrimSpace(p.Name) == "" {
			continue
		}
		params[p.Name] = parseParameterValue(p.Value)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	out, err := client.TriggerPipeline(cfg.ProjectSlug, TriggerPipelineRequest{
		Branch:     strings.TrimSpace(cfg.Branch),
		Tag:        strings.TrimSpace(cfg.Tag),
		Parameters: params,
	})
	if err != nil {
		return err
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"circleci.pipeline.triggered",
		[]any{out},
	)
}

func (t *TriggerPipeline) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 200, nil
}

func (t *TriggerPipeline) Actions() []core.Action { return []core.Action{} }
func (t *TriggerPipeline) HandleAction(ctx core.ActionContext) error {
	return nil
}
func (t *TriggerPipeline) Cancel(ctx core.ExecutionContext) error { return nil }
func (t *TriggerPipeline) Cleanup(ctx core.SetupContext) error    { return nil }
