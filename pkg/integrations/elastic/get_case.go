package elastic

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type GetCase struct{}

type GetCaseNodeMetadata struct {
	CaseName string `json:"caseName,omitempty" mapstructure:"caseName"`
}

type GetCaseConfiguration struct {
	CaseID string `json:"caseId" mapstructure:"caseId"`
}

func (c *GetCase) Name() string  { return "elastic.getCase" }
func (c *GetCase) Label() string { return "Get Case" }
func (c *GetCase) Description() string {
	return "Retrieve a Kibana Security case by ID"
}
func (c *GetCase) Icon() string  { return "elastic" }
func (c *GetCase) Color() string { return "gray" }

func (c *GetCase) Documentation() string {
	return `The Get Case component retrieves an existing case from Kibana Security by its ID.

## Configuration

- **Case**: The Kibana case to retrieve

## Outputs

The component emits an event containing:
- ` + "`id`" + `: The case ID
- ` + "`title`" + `: The case title
- ` + "`description`" + `: The case description
- ` + "`status`" + `: The case status
- ` + "`severity`" + `: The case severity
- ` + "`tags`" + `: The case tags
- ` + "`version`" + `: The current case version returned by Kibana
- ` + "`createdAt`" + `: The timestamp when the case was created
- ` + "`updatedAt`" + `: The timestamp when the case was last updated`
}

func (c *GetCase) OutputChannels(_ any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetCase) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "caseId",
			Label:       "Case",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The Kibana case to retrieve.",
			Placeholder: "Select a case",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeCase,
				},
			},
		},
	}
}

func (c *GetCase) Setup(ctx core.SetupContext) error {
	var config GetCaseConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if strings.TrimSpace(config.CaseID) == "" {
		return fmt.Errorf("caseId is required")
	}

	return c.resolveMetadata(ctx, config.CaseID)
}

func (c *GetCase) resolveMetadata(ctx core.SetupContext, caseID string) error {
	meta := GetCaseNodeMetadata{}
	if strings.Contains(caseID, "{{") {
		meta.CaseName = caseID
	} else {
		client, err := NewClient(ctx.HTTP, ctx.Integration)
		if err != nil {
			return fmt.Errorf("failed to create Elastic client: %w", err)
		}
		caseResp, err := client.GetCase(caseID)
		if err != nil {
			return fmt.Errorf("failed to get case: %w", err)
		}
		meta.CaseName = caseResp.Title
	}
	return ctx.Metadata.Set(meta)
}

func (c *GetCase) Execute(ctx core.ExecutionContext) error {
	var config GetCaseConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to decode configuration: %v", err))
	}

	config.CaseID = strings.TrimSpace(config.CaseID)
	if config.CaseID == "" {
		return ctx.ExecutionState.Fail("error", "caseId is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to create Elastic client: %v", err))
	}

	resp, err := client.GetCase(config.CaseID)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to get case: %v", err))
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"elastic.case.retrieved",
		[]any{map[string]any{
			"id":          resp.ID,
			"title":       resp.Title,
			"description": resp.Description,
			"status":      resp.Status,
			"severity":    resp.Severity,
			"tags":        resp.Tags,
			"version":     resp.Version,
			"createdAt":   resp.CreatedAt,
			"updatedAt":   resp.UpdatedAt,
		}},
	)
}

func (c *GetCase) Actions() []core.Action                  { return nil }
func (c *GetCase) HandleAction(_ core.ActionContext) error { return nil }
func (c *GetCase) Cancel(_ core.ExecutionContext) error    { return nil }
func (c *GetCase) Cleanup(_ core.SetupContext) error       { return nil }
func (c *GetCase) HandleWebhook(_ core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}
func (c *GetCase) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}
