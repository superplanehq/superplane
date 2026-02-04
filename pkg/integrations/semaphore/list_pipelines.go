package semaphore

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type ListPipelines struct{}

type ListPipelinesConfiguration struct {
	Project       string `json:"project" mapstructure:"project"`
	BranchName    string `json:"branchName,omitempty" mapstructure:"branchName"`
	YmlFilePath   string `json:"ymlFilePath,omitempty" mapstructure:"ymlFilePath"`
	CreatedAfter  string `json:"createdAfter,omitempty" mapstructure:"createdAfter"`
	CreatedBefore string `json:"createdBefore,omitempty" mapstructure:"createdBefore"`
	DoneAfter     string `json:"doneAfter,omitempty" mapstructure:"doneAfter"`
	DoneBefore    string `json:"doneBefore,omitempty" mapstructure:"doneBefore"`
	Limit         *int   `json:"limit,omitempty" mapstructure:"limit"`
}

func (c *ListPipelines) Name() string {
	return "semaphore.listPipelines"
}

func (c *ListPipelines) Label() string {
	return "List Pipelines"
}

func (c *ListPipelines) Description() string {
	return "Lists pipelines for a Semaphore project with optional filters."
}

func (c *ListPipelines) Documentation() string {
	return "https://docs.semaphoreci.com/reference/api-v1alpha/#list-pipelines"
}

func (c *ListPipelines) Icon() string {
	return "list"
}

func (c *ListPipelines) Color() string {
	return "gray"
}

func (c *ListPipelines) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *ListPipelines) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "project",
			Type:        configuration.FieldTypeString,
			Label:       "Project",
			Description: "Semaphore project ID or name",
			Required:    true,
		},
		{
			Name:        "branchName",
			Type:        configuration.FieldTypeString,
			Label:       "Branch Name",
			Description: "Filter pipelines by branch name",
			Required:    false,
		},
		{
			Name:        "ymlFilePath",
			Type:        configuration.FieldTypeString,
			Label:       "YAML File Path",
			Description: "Filter pipelines by pipeline YAML file path",
			Required:    false,
		},
		{
			Name:        "createdAfter",
			Type:        configuration.FieldTypeString,
			Label:       "Created After",
			Description: "Filter pipelines created after this date (ISO 8601 or Unix timestamp)",
			Required:    false,
		},
		{
			Name:        "createdBefore",
			Type:        configuration.FieldTypeString,
			Label:       "Created Before",
			Description: "Filter pipelines created before this date (ISO 8601 or Unix timestamp)",
			Required:    false,
		},
		{
			Name:        "doneAfter",
			Type:        configuration.FieldTypeString,
			Label:       "Done After",
			Description: "Filter pipelines completed after this date (ISO 8601 or Unix timestamp)",
			Required:    false,
		},
		{
			Name:        "doneBefore",
			Type:        configuration.FieldTypeString,
			Label:       "Done Before",
			Description: "Filter pipelines completed before this date (ISO 8601 or Unix timestamp)",
			Required:    false,
		},
		{
			Name:        "limit",
			Type:        configuration.FieldTypeNumber,
			Label:       "Limit",
			Description: "Maximum number of pipelines to return (max 100)",
			Required:    false,
		},
	}
}

func (c *ListPipelines) Setup(ctx core.SetupContext) error {
	var config ListPipelinesConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Project == "" {
		return fmt.Errorf("project is required")
	}

	if config.Limit != nil && (*config.Limit < 1 || *config.Limit > 100) {
		return fmt.Errorf("limit must be between 1 and 100")
	}

	return nil
}

func (c *ListPipelines) Execute(ctx core.ExecutionContext) error {
	var config ListPipelinesConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create Semaphore client: %w", err)
	}

	// Resolve project ID if name was provided
	project, err := client.GetProject(config.Project)
	if err != nil {
		return fmt.Errorf("failed to get project: %w", err)
	}

	// Build query parameters
	params := url.Values{}
	params.Set("project_id", project.Metadata.ProjectID)

	if config.BranchName != "" {
		params.Set("branch_name", config.BranchName)
	}
	if config.YmlFilePath != "" {
		params.Set("yml_file_path", config.YmlFilePath)
	}
	if config.CreatedAfter != "" {
		params.Set("created_after", config.CreatedAfter)
	}
	if config.CreatedBefore != "" {
		params.Set("created_before", config.CreatedBefore)
	}
	if config.DoneAfter != "" {
		params.Set("done_after", config.DoneAfter)
	}
	if config.DoneBefore != "" {
		params.Set("done_before", config.DoneBefore)
	}
	if config.Limit != nil {
		params.Set("page_size", strconv.Itoa(*config.Limit))
	}

	pipelines, err := client.ListPipelinesWithParams(params)
	if err != nil {
		return fmt.Errorf("failed to list pipelines: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"semaphore.pipelines",
		[]any{pipelines},
	)
}

func (c *ListPipelines) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *ListPipelines) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 200, nil
}

func (c *ListPipelines) Actions() []core.Action {
	return []core.Action{}
}

func (c *ListPipelines) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *ListPipelines) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *ListPipelines) Cleanup(ctx core.SetupContext) error {
	return nil
}
