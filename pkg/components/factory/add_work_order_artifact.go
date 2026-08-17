package factory

import (
	"net/http"

	"github.com/go-viper/mapstructure/v2"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

const AddWorkOrderArtifactComponentName = "addWorkOrderArtifact"

func init() {
	registry.RegisterAction(AddWorkOrderArtifactComponentName, &AddWorkOrderArtifact{})
}

type AddWorkOrderArtifact struct{}

// ArtifactDataEntry is one {name, value} row in the free-form data
// list; flattened into `map[string]any` at Execute time.
type ArtifactDataEntry struct {
	Name  string `json:"name" mapstructure:"name"`
	Value string `json:"value" mapstructure:"value"`
}

type AddWorkOrderArtifactConfiguration struct {
	OrderID      string              `json:"orderId" mapstructure:"orderId"`
	ArtifactType string              `json:"artifactType" mapstructure:"artifactType"`
	URL          string              `json:"url" mapstructure:"url"`
	Number       string              `json:"number" mapstructure:"number"`
	State        string              `json:"state" mapstructure:"state"`
	Title        string              `json:"title" mapstructure:"title"`
	Body         string              `json:"body" mapstructure:"body"`
	Name         string              `json:"name" mapstructure:"name"`
	ArtifactKey  string              `json:"artifactKey" mapstructure:"artifactKey"`
	Data         []ArtifactDataEntry `json:"data" mapstructure:"data"`
}

func (c *AddWorkOrderArtifact) Name() string {
	return AddWorkOrderArtifactComponentName
}

func (c *AddWorkOrderArtifact) Label() string {
	return "Add Work Order Artifact"
}

func (c *AddWorkOrderArtifact) Description() string {
	return "Attach a typed artifact (PR, markdown note, or branch) to a work order"
}

func (c *AddWorkOrderArtifact) Documentation() string {
	return `The Add Work Order Artifact component stores a typed artifact against a work order.

Supported types:

- **Pull request** (` + "`pr`" + `): requires ` + "`url`" + `; optional ` + "`number`" + `, ` + "`title`" + `, and ` + "`state`" + ` (` + "`open`" + `/` + "`draft`" + `/` + "`closed`" + `/` + "`merged`" + `, defaults to ` + "`open`" + `) which drives the icon/color of the artifact chip in the work order UI.
- **Markdown note** (` + "`markdown`" + `): requires ` + "`body`" + `; optional ` + "`title`" + `.
- **Branch** (` + "`branch`" + `): requires ` + "`name`" + ` (the branch name); optional ` + "`url`" + ` to link to the branch on its provider (e.g. a GitHub tree URL).

PR and markdown types accept a free-form ` + "`data`" + ` list of ` + "`{name, value}`" + ` entries that gets merged into the artifact's ` + "`data`" + ` map. Typed inputs take precedence over free-form entries with the same key.

Set ` + "`artifactKey`" + ` to tag the artifact with a queryable key (e.g. the pull request's URL) so a later ` + "`findWorkOrder`" + ` (` + "`by: artifactKey`" + `) step can resolve this work order from it — useful in flows that aren't dispatched from a factory line, such as closing a work order from a ` + "`github.onPullRequest`" + ` merged event. Keys are unique per factory.

A pull request's ` + "`state`" + ` normally changes after it's attached — set ` + "`artifactKey`" + ` at attach time, then use ` + "`updateWorkOrderArtifact`" + ` (targeting the same key) from a ` + "`github.onPullRequest`" + ` flow to keep it current as the PR is drafted, reopened, closed, or merged.

` + "`orderId`" + ` explicitly targets the work order — it defaults to ` + "`{{ order().id }}`" + `, the work order driving the current run, which only resolves when the flow was dispatched from a factory line. In a flow triggered by an external event, replace it with e.g. ` + "`{{ previous().data.workOrder.id }}`" + `. This component can only be used in factory-owned apps.`
}

func (c *AddWorkOrderArtifact) Icon() string {
	return "factory"
}

func (c *AddWorkOrderArtifact) Color() string {
	return "blue"
}

func (c *AddWorkOrderArtifact) ExampleOutput() map[string]any {
	return map[string]any{
		"timestamp": "2026-01-01T00:00:00Z",
		"type":      "workOrder.artifactAdded",
		"data": map[string]any{
			"artifact": map[string]any{
				"id":   "art-123",
				"type": "pr",
				"data": map[string]any{
					"url":      "https://github.com/example/repo/pull/42",
					"number":   "42",
					"title":    "Draft implementation",
					"state":    "open",
					"provider": "github",
				},
			},
		},
	}
}

func (c *AddWorkOrderArtifact) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *AddWorkOrderArtifact) Configuration() []configuration.Field {
	prOnly := []configuration.VisibilityCondition{{Field: "artifactType", Values: []string{"pr"}}}
	markdownOnly := []configuration.VisibilityCondition{{Field: "artifactType", Values: []string{"markdown"}}}
	branchOnly := []configuration.VisibilityCondition{{Field: "artifactType", Values: []string{"branch"}}}
	linkableTypes := []configuration.VisibilityCondition{{Field: "artifactType", Values: []string{"pr", "branch"}}}
	bothTypes := []configuration.VisibilityCondition{{Field: "artifactType", Values: []string{"pr", "markdown"}}}
	withMetadata := []configuration.VisibilityCondition{{Field: "artifactType", Values: []string{"pr", "markdown"}}}

	return []configuration.Field{
		{
			Name:        "orderId",
			Label:       "Work Order ID",
			Description: "Work order to target. Defaults to the work order driving the current run (only resolves when this flow was dispatched from a factory line). Replace it with e.g. {{ previous().data.workOrder.id }} otherwise.",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Default:     "{{ order().id }}",
		},
		{
			Name:        "artifactType",
			Label:       "Type",
			Description: "The kind of artifact to store",
			Type:        configuration.FieldTypeSelect,
			Required:    true,
			Default:     "pr",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Pull Request", Value: "pr"},
						{Label: "Markdown", Value: "markdown"},
						{Label: "Branch", Value: "branch"},
					},
				},
			},
		},
		{
			Name:                 "url",
			Label:                "URL",
			Description:          "Link to the pull request or branch (must be http or https). Required for pull requests; optional for branches — e.g. a GitHub tree URL like https://github.com/{owner}/{repo}/tree/{branch}.",
			Type:                 configuration.FieldTypeString,
			Required:             false,
			VisibilityConditions: linkableTypes,
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "artifactType", Values: []string{"pr"}},
			},
		},
		{
			Name:                 "number",
			Label:                "Number",
			Description:          "Optional pull request number (rendered as #<n>)",
			Type:                 configuration.FieldTypeString,
			Required:             false,
			VisibilityConditions: prOnly,
		},
		{
			Name:                 "state",
			Label:                "State",
			Description:          "Pull request state — drives the artifact chip's icon/color. Update it later with updateWorkOrderArtifact as the PR progresses.",
			Type:                 configuration.FieldTypeSelect,
			Required:             false,
			Default:              "open",
			VisibilityConditions: prOnly,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Open", Value: "open"},
						{Label: "Draft", Value: "draft"},
						{Label: "Closed", Value: "closed"},
						{Label: "Merged", Value: "merged"},
					},
				},
			},
		},
		{
			Name:                 "body",
			Label:                "Body",
			Description:          "Markdown note body — rendered inline in the work order timeline",
			Type:                 configuration.FieldTypeText,
			Required:             false,
			VisibilityConditions: markdownOnly,
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "artifactType", Values: []string{"markdown"}},
			},
		},
		{
			Name:                 "name",
			Label:                "Name",
			Description:          "Branch name (e.g. feature/refund-retry)",
			Type:                 configuration.FieldTypeString,
			Required:             false,
			VisibilityConditions: branchOnly,
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "artifactType", Values: []string{"branch"}},
			},
		},
		{
			Name:                 "title",
			Label:                "Title",
			Description:          "Optional artifact title",
			Type:                 configuration.FieldTypeString,
			Required:             false,
			VisibilityConditions: bothTypes,
		},
		{
			Name:        "artifactKey",
			Label:       "Artifact Key",
			Description: "Optional queryable key for this artifact (e.g. a pull request's URL), unique per factory. Lets findWorkOrder (by: artifactKey) resolve this work order later.",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Togglable:   true,
			Default:     "",
		},
		{
			Name:                 "data",
			Label:                "Metadata",
			Description:          "Extra name/value pairs merged into the artifact's data map (typed fields above take precedence on name collisions)",
			Type:                 configuration.FieldTypeList,
			Required:             false,
			VisibilityConditions: withMetadata,
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Entry",
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

func (c *AddWorkOrderArtifact) Execute(ctx core.ExecutionContext) error {
	config := AddWorkOrderArtifactConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return err
	}

	data := buildArtifactData(config)

	artifact, err := ctx.Factory.AddWorkOrderArtifact(core.AddWorkOrderArtifactParams{
		OrderID: config.OrderID,
		Type:    config.ArtifactType,
		Data:    data,
		Key:     config.ArtifactKey,
	})
	if err != nil {
		return err
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"workOrder.artifactAdded",
		[]any{map[string]any{
			"artifact": artifact,
		}},
	)
}

func (c *AddWorkOrderArtifact) Setup(ctx core.SetupContext) error {
	return nil
}

func (c *AddWorkOrderArtifact) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *AddWorkOrderArtifact) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *AddWorkOrderArtifact) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *AddWorkOrderArtifact) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *AddWorkOrderArtifact) HandleHook(ctx core.ActionHookContext) error {
	return nil
}

// buildArtifactData folds the free-form list into a map and layers the
// typed inputs on top, so a user who defines both `url` and a `url`
// row still ends up with the typed value on the wire.
func buildArtifactData(config AddWorkOrderArtifactConfiguration) map[string]any {
	data := artifactDataToMap(config.Data)

	typed := map[string]string{
		"url":    config.URL,
		"number": config.Number,
		"state":  config.State,
		"title":  config.Title,
		"body":   config.Body,
		"name":   config.Name,
	}

	for key, value := range typed {
		if value == "" {
			continue
		}
		if data == nil {
			data = map[string]any{}
		}
		data[key] = value
	}

	return data
}

// artifactDataToMap flattens the list into a map; blank names are
// skipped and duplicate names take the last value written.
func artifactDataToMap(entries []ArtifactDataEntry) map[string]any {
	if len(entries) == 0 {
		return nil
	}

	result := make(map[string]any, len(entries))
	for _, entry := range entries {
		if entry.Name == "" {
			continue
		}
		result[entry.Name] = entry.Value
	}

	return result
}
