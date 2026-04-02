package grafana

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type ListAlertRules struct{}

type ListAlertRulesSpec struct {
	FolderUID string `json:"folderUID,omitempty" mapstructure:"folderUID"`
	Group     string `json:"group,omitempty" mapstructure:"group"`
}

type ListAlertRulesNodeMetadata struct {
	FolderTitle string `json:"folderTitle,omitempty" mapstructure:"folderTitle"`
}

type ListAlertRulesOutput struct {
	AlertRules []AlertRuleSummary `json:"alertRules" mapstructure:"alertRules"`
}

func decodeListAlertRulesSpec(input any) (ListAlertRulesSpec, error) {
	spec := ListAlertRulesSpec{}
	if err := decodeAlertRuleSpec(input, &spec); err != nil {
		return ListAlertRulesSpec{}, fmt.Errorf("error decoding configuration: %v", err)
	}
	spec.FolderUID = strings.TrimSpace(spec.FolderUID)
	spec.Group = strings.TrimSpace(spec.Group)
	return spec, nil
}

func (c *ListAlertRules) Name() string {
	return "grafana.listAlertRules"
}

func (c *ListAlertRules) Label() string {
	return "List Alert Rules"
}

func (c *ListAlertRules) Description() string {
	return "List Grafana-managed alert rules for the connected Grafana instance"
}

func (c *ListAlertRules) Documentation() string {
	return `The List Alert Rules component lists Grafana-managed alert rules using the Alerting Provisioning HTTP API.

## Use Cases

- **Alert audits**: review which Grafana alert rules currently exist
- **Workflow enrichment**: send alert inventories to Slack, Jira, or documentation steps
- **Follow-up automation**: feed alert rule summaries into downstream review or cleanup workflows

## Configuration

All fields are optional:

- **Folder**: When set, only alert rules in this Grafana folder are listed
- **Rule Group**: When set, only rules in this Grafana rule group are listed

When both are omitted, the component lists alert rules across the instance (subject to Grafana permissions).

## Output

Returns an object containing the list of Grafana alert rule summaries, including each rule UID and title.`
}

func (c *ListAlertRules) Icon() string {
	return "bell"
}

func (c *ListAlertRules) Color() string {
	return "blue"
}

func (c *ListAlertRules) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *ListAlertRules) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "folderUID",
			Label:       "Folder",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Limit results to alert rules in this folder",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: resourceTypeFolder,
				},
			},
		},
		{
			Name:        "group",
			Label:       "Rule Group",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Limit results to alert rules in this rule group",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: resourceTypeRuleGroup,
				},
			},
		},
	}
}

func (c *ListAlertRules) Setup(ctx core.SetupContext) error {
	spec, err := decodeListAlertRulesSpec(ctx.Configuration)
	if err != nil {
		return err
	}

	if spec.FolderUID == "" || ctx.Metadata == nil || ctx.HTTP == nil {
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration, true)
	if err != nil {
		return nil
	}

	folders, err := client.ListFolders()
	if err != nil {
		return nil
	}

	for _, folder := range folders {
		if strings.TrimSpace(folder.UID) != spec.FolderUID {
			continue
		}

		title := strings.TrimSpace(folder.Title)
		if title != "" {
			_ = ctx.Metadata.Set(ListAlertRulesNodeMetadata{FolderTitle: title})
		}

		break
	}

	return nil
}

func (c *ListAlertRules) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeListAlertRulesSpec(ctx.Configuration)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration, true)
	if err != nil {
		return fmt.Errorf("error creating client: %w", err)
	}

	rules, err := client.ListAlertRules(spec.FolderUID, spec.Group)
	if err != nil {
		return fmt.Errorf("error listing alert rules: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"grafana.alertRules",
		[]any{ListAlertRulesOutput{
			AlertRules: rules,
		}},
	)
}

func (c *ListAlertRules) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *ListAlertRules) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *ListAlertRules) Actions() []core.Action {
	return []core.Action{}
}

func (c *ListAlertRules) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *ListAlertRules) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *ListAlertRules) Cleanup(ctx core.SetupContext) error {
	return nil
}
