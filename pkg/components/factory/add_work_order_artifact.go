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
	OrderID      string `json:"orderId" mapstructure:"orderId"`
	ArtifactType string `json:"artifactType" mapstructure:"artifactType"`
	URL          string `json:"url" mapstructure:"url"`
	Number       string `json:"number" mapstructure:"number"`
	// State / Merged / Draft accept expressions, so a flow can wire
	// them directly to a `github.onPullRequest` payload. `any` because
	// after resolution the value may be a bool, string, or number.
	State       any                 `json:"state,omitempty" mapstructure:"state,omitempty"`
	Merged      any                 `json:"merged,omitempty" mapstructure:"merged,omitempty"`
	Draft       any                 `json:"draft,omitempty" mapstructure:"draft,omitempty"`
	Title       string              `json:"title" mapstructure:"title"`
	Body        string              `json:"body" mapstructure:"body"`
	Name        string              `json:"name" mapstructure:"name"`
	Repository  string              `json:"repository" mapstructure:"repository"`
	ArtifactKey string              `json:"artifactKey" mapstructure:"artifactKey"`
	MergedAt    string              `json:"mergedAt" mapstructure:"mergedAt"`
	ClosedAt    string              `json:"closedAt" mapstructure:"closedAt"`
	Data        []ArtifactDataEntry `json:"data" mapstructure:"data"`
}

func (c *AddWorkOrderArtifact) Name() string {
	return AddWorkOrderArtifactComponentName
}

func (c *AddWorkOrderArtifact) Label() string {
	return "Add Work Order Artifact"
}

func (c *AddWorkOrderArtifact) Description() string {
	return "Attach a typed artifact (PR, markdown note, branch, or link) to a work order"
}

func (c *AddWorkOrderArtifact) Documentation() string {
	return `The Add Work Order Artifact component stores a typed artifact against a work order.

Supported types:

- **Pull request** (` + "`pr`" + `): requires ` + "`url`" + `; optional ` + "`number`" + `, ` + "`title`" + `, ` + "`state`" + `, ` + "`merged`" + `, ` + "`draft`" + `, ` + "`mergedAt`" + `, and ` + "`closedAt`" + `. The ` + "`state`" + ` field (` + "`open`" + `/` + "`draft`" + `/` + "`closed`" + `/` + "`merged`" + `, defaults to ` + "`open`" + `) drives the icon/color of the artifact chip in the work order UI. ` + "`state`" + `, ` + "`merged`" + `, and ` + "`draft`" + ` all accept expressions, so a flow can wire them straight to a ` + "`github.onPullRequest`" + ` payload: a GitHub-shaped ` + "`state: \"closed\"`" + ` + ` + "`merged: true`" + ` folds into SuperPlane's ` + "`state: \"merged\"`" + ` before it hits the artifact. Set ` + "`mergedAt`" + ` / ` + "`closedAt`" + ` (RFC3339, usually from the GitHub event) when you attach an already-merged or closed PR so Velocity uses the real day instead of now.
- **Markdown note** (` + "`markdown`" + `): requires ` + "`body`" + `; optional ` + "`title`" + `.
- **Branch** (` + "`branch`" + `): requires ` + "`name`" + ` and ` + "`repository`" + ` (` + "`owner/repo`" + ` or a repository http(s) URL), or an explicit ` + "`url`" + `. SuperPlane writes a GitHub tree URL from the repository and branch name at attach time and does not wait for a pull request.
- **Link** (` + "`link`" + `): requires ` + "`url`" + ` (must be http or https); optional ` + "`title`" + ` for the artifact chip's label — e.g. attach a preview-environment URL as "Preview".

PR, markdown, and link types accept a free-form ` + "`data`" + ` list of ` + "`{name, value}`" + ` entries that gets merged into the artifact's ` + "`data`" + ` map. Typed inputs take precedence over free-form entries with the same key.

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
	linkableTypes := []configuration.VisibilityCondition{{Field: "artifactType", Values: []string{"pr", "branch", "link"}}}
	titledTypes := []configuration.VisibilityCondition{{Field: "artifactType", Values: []string{"pr", "markdown", "link"}}}
	withMetadata := []configuration.VisibilityCondition{{Field: "artifactType", Values: []string{"pr", "markdown", "link"}}}

	fields := []configuration.Field{
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
						{Label: "Link", Value: "link"},
					},
				},
			},
		},
		{
			Name:                 "url",
			Label:                "URL",
			Description:          "Link to the pull request, branch, or external resource (must be http or https). Required for pull requests and links. For branches, optional: when empty, SuperPlane writes a GitHub tree URL from repository and name. Example: https://github.com/{owner}/{repo}/tree/{branch}.",
			Type:                 configuration.FieldTypeString,
			Required:             false,
			VisibilityConditions: linkableTypes,
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "artifactType", Values: []string{"pr", "link"}},
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
	}

	fields = append(fields, prArtifactLifecycleFields(prArtifactLifecycleFieldOptions{
		Visibility:   prOnly,
		StateDefault: "open",
	})...)

	return append(fields,
		configuration.Field{
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
		configuration.Field{
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
		configuration.Field{
			Name:                 "repository",
			Label:                "Repository",
			Description:          "Repository that owns the branch (`owner/repo` or the repository https URL). Required when URL is empty. SuperPlane writes a GitHub tree URL from this value and the branch name.",
			Type:                 configuration.FieldTypeString,
			Required:             false,
			VisibilityConditions: branchOnly,
		},
		configuration.Field{
			Name:                 "title",
			Label:                "Title",
			Description:          "Optional artifact title — for links, this becomes the chip's label (e.g. \"Preview\")",
			Type:                 configuration.FieldTypeString,
			Required:             false,
			VisibilityConditions: titledTypes,
		},
		configuration.Field{
			Name:        "artifactKey",
			Label:       "Artifact Key",
			Description: "Optional queryable key for this artifact (e.g. a pull request's URL), unique per factory. Lets findWorkOrder (by: artifactKey) resolve this work order later.",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Togglable:   true,
			Default:     "",
		},
		configuration.Field{
			Name:                 "mergedAt",
			Label:                "Merged At",
			Description:          "Optional RFC3339 merge timestamp — usually {{ event.data.pull_request.merged_at }}. Set when attaching an already-merged PR so Velocity attributes it to the real merge day; unset falls back to now.",
			Type:                 configuration.FieldTypeString,
			Required:             false,
			Togglable:            true,
			VisibilityConditions: prOnly,
		},
		configuration.Field{
			Name:                 "closedAt",
			Label:                "Closed At",
			Description:          "Optional RFC3339 close timestamp — usually {{ event.data.pull_request.closed_at }}. Set when attaching an already-closed PR so Velocity waste attributes it to the real close day.",
			Type:                 configuration.FieldTypeString,
			Required:             false,
			Togglable:            true,
			VisibilityConditions: prOnly,
		},
		configuration.Field{
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
	)
}

func (c *AddWorkOrderArtifact) ValidateNodeConfiguration(config map[string]any) error {
	decoded := AddWorkOrderArtifactConfiguration{}
	if err := mapstructure.Decode(config, &decoded); err != nil {
		return err
	}
	return validateBranchArtifactConfiguration(decoded)
}

func (c *AddWorkOrderArtifact) Execute(ctx core.ExecutionContext) error {
	config := AddWorkOrderArtifactConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return err
	}

	data, err := buildArtifactData(config)
	if err != nil {
		return err
	}

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
func buildArtifactData(config AddWorkOrderArtifactConfiguration) (map[string]any, error) {
	data := artifactDataToMap(config.Data)

	typed := map[string]string{
		"url":      config.URL,
		"number":   config.Number,
		"title":    config.Title,
		"body":     config.Body,
		"name":     config.Name,
		"mergedAt": config.MergedAt,
		"closedAt": config.ClosedAt,
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

	data = applyBranchTreeURL(config, data)
	if err := requireReachableBranchURL(config.ArtifactType, data); err != nil {
		return nil, err
	}

	if config.ArtifactType != "pr" {
		return data, nil
	}

	updates, err := prArtifactStateUpdates(config.State, config.Merged, config.Draft)
	if err != nil {
		return nil, err
	}
	if len(updates) > 0 {
		data = ensureArtifactData(data)
		for key, value := range updates {
			data[key] = value
		}
	}

	return data, nil
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
