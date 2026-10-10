package issues

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
)

type OnIssue struct{}

const (
	issueLabelFilterExclude   = "exclude"
	issueAssignmentAssigned   = "assigned"
	issueAssignmentUnassigned = "unassigned"
	issueSuperplaneLabel      = "superplane"
	issuePermissionNone       = "none"
)

type OnIssueConfiguration struct {
	Repository           string   `json:"repository" mapstructure:"repository"`
	Actions              []string `json:"actions" mapstructure:"actions"`
	Labels               []string `json:"labels" mapstructure:"labels"`
	LabelFilterMode      string   `json:"labelFilterMode" mapstructure:"labelFilterMode"`
	Assignment           string   `json:"assignment" mapstructure:"assignment"`
	AuthorsWithAccess    bool     `json:"authorsWithAccess" mapstructure:"authorsWithAccess"`
	SuperplaneLabelAdded bool     `json:"superplaneLabelAdded" mapstructure:"superplaneLabelAdded"`
}

func (i *OnIssue) Name() string {
	return "github.onIssue"
}

func (i *OnIssue) Label() string {
	return "On Issue"
}

func (i *OnIssue) Description() string {
	return "Listen to issue events"
}

func (i *OnIssue) Documentation() string {
	return `The On Issue trigger starts a workflow execution when issue events occur in a GitHub repository.

## Use Cases

- **Issue automation**: Automate responses to new or updated issues
- **Notification workflows**: Send notifications when issues are created or closed
- **Task management**: Sync issues with external task management systems
- **Label automation**: Automatically label or categorize issues

## Configuration

- **Repository**: Select the GitHub repository to monitor
- **Actions**: Select which issue actions to listen for (opened, closed, reopened, etc.)
- **Labels**: Optional. Start a run only when the issue has one of these labels. Leave empty to accept every label.
- **Label filter**: Optional. Include starts a run when the issue has one of the labels. Exclude starts a run when the issue has none of those labels.
- **Assignment**: Optional. Start a run only for assigned issues, only for unassigned issues, or for any assignment.
- **Author is a repository collaborator**: Optional. Start a run only when the author can access the repository.
- **The "superplane" label is added to the issue**: Optional. For a labeled event, start a run only when that label is added to an open issue.

## Event Data

Each issue event includes:
- **action**: The action that triggered the event (opened, closed, reopened, etc.)
- **issue**: Complete issue information including title, body, state, labels, assignees
- **repository**: Repository information
- **sender**: User who triggered the event

## Webhook Setup

This trigger automatically sets up a GitHub webhook when configured. The webhook is managed by SuperPlane and will be cleaned up when the trigger is removed.`
}

func (i *OnIssue) Icon() string {
	return "github"
}

func (i *OnIssue) Color() string {
	return "gray"
}

func (i *OnIssue) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "repository",
			Label:    "Repository",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "repository",
					UseNameAsValue: true,
				},
			},
		},
		{
			Name:     "actions",
			Label:    "Actions",
			Type:     configuration.FieldTypeMultiSelect,
			Required: true,
			Default:  []string{"opened"},
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Opened", Value: "opened"},
						{Label: "Edited", Value: "edited"},
						{Label: "Deleted", Value: "deleted"},
						{Label: "Transferred", Value: "transferred"},
						{Label: "Pinned", Value: "pinned"},
						{Label: "Unpinned", Value: "unpinned"},
						{Label: "Closed", Value: "closed"},
						{Label: "Reopened", Value: "reopened"},
						{Label: "Assigned", Value: "assigned"},
						{Label: "Unassigned", Value: "unassigned"},
						{Label: "Labeled", Value: "labeled"},
						{Label: "Unlabeled", Value: "unlabeled"},
						{Label: "Locked", Value: "locked"},
						{Label: "Unlocked", Value: "unlocked"},
						{Label: "Milestoned", Value: "milestoned"},
						{Label: "Demilestoned", Value: "demilestoned"},
					},
				},
			},
		},
		{
			Name:        "labels",
			Label:       "Issue has one of these labels",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Description: "Start a run only when the issue has one of these labels.",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Label",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeString,
					},
				},
			},
		},
		{
			Name:        "labelFilterMode",
			Label:       "Label filter",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Default:     "include",
			Description: "Include starts a run when the issue has one of the labels. Exclude starts a run when the issue has none of those labels.",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Include", Value: "include"},
						{Label: "Exclude", Value: "exclude"},
					},
				},
			},
		},
		{
			Name:        "assignment",
			Label:       "Assignment",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Default:     "any",
			Description: "Start a run only for issues with this assignment.",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Any assignment", Value: "any"},
						{Label: "Issue is assigned", Value: "assigned"},
						{Label: "Issue is unassigned", Value: "unassigned"},
					},
				},
			},
		},
		{
			Name:        "authorsWithAccess",
			Label:       "Author is a repository collaborator",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     false,
			Description: "Start a run only when the author can access the repository.",
		},
		{
			Name:        "superplaneLabelAdded",
			Label:       `The "superplane" label is added to the issue`,
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     false,
			Description: "For a labeled event, start a run only when the superplane label is added to an open issue.",
		},
	}
}

func (i *OnIssue) Setup(ctx core.TriggerContext) error {
	err := common.EnsureRepoInMetadata(
		ctx.Metadata,
		ctx.Integration,
		ctx.HTTP,
		ctx.Configuration,
	)

	if err != nil {
		return err
	}

	var config OnIssueConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	return ctx.Integration.RequestWebhook(common.WebhookConfiguration{
		EventType:  "issues",
		Repository: config.Repository,
	})
}

func (i *OnIssue) Hooks() []core.Hook {
	return []core.Hook{}
}

func (i *OnIssue) HandleHook(ctx core.TriggerHookContext) (map[string]any, error) {
	return nil, nil
}

func (i *OnIssue) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	ctx = common.WithWebhookLogger(ctx, i.Name())
	ctx.Logger.Infof("Received GitHub webhook")

	config := OnIssueConfiguration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		ctx.Logger.Errorf("Failed to decode configuration: %v", err)
		return http.StatusInternalServerError, nil, fmt.Errorf("failed to decode configuration: %w", err)
	}

	eventType := ctx.Headers.Get("X-GitHub-Event")
	if eventType == "" {
		ctx.Logger.Errorf("Missing X-GitHub-Event header")
		return http.StatusBadRequest, nil, fmt.Errorf("missing X-GitHub-Event header")
	}

	if eventType != "issues" {
		ctx.Logger.Infof("Ignoring event - event type %q is not a issues event", eventType)
		return http.StatusOK, nil, nil
	}

	code, err := common.VerifySignature(ctx)
	if err != nil {
		ctx.Logger.Errorf("Failed to verify signature: %v", err)
		return code, nil, err
	}

	data := map[string]any{}
	err = json.Unmarshal(ctx.Body, &data)
	if err != nil {
		ctx.Logger.Errorf("Failed to parse request body: %v", err)
		return http.StatusBadRequest, nil, fmt.Errorf("error parsing request body: %v", err)
	}

	if !common.WhitelistedAction(data, config.Actions) {
		action, ok := common.ExtractAction(data)
		if !ok {
			ctx.Logger.Info("Ignoring event - without a valid action")
			return http.StatusOK, nil, nil
		}

		ctx.Logger.Infof("Ignoring event - action %q is not configured", action)
		return http.StatusOK, nil, nil
	}

	if !issueMatchesFilters(data, config) {
		ctx.Logger.Info("Ignoring event - issue does not match the configured filters")
		return http.StatusOK, nil, nil
	}

	if config.AuthorsWithAccess {
		allowed, code, accessErr := authorHasRepositoryAccess(ctx, config.Repository, issueAuthorLogin(data))
		if accessErr != nil || !allowed {
			return code, nil, accessErr
		}
	}

	blankMissingIssueBody(data)
	err = ctx.Events.Emit("github.issue", data)
	if err != nil {
		ctx.Logger.Errorf("Failed to emit event: %v", err)
		return http.StatusInternalServerError, nil, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil, nil
}

func (i *OnIssue) Cleanup(ctx core.TriggerContext) error {
	return nil
}

func blankMissingIssueBody(data map[string]any) {
	issue, ok := data["issue"].(map[string]any)
	if !ok {
		return
	}

	body, present := issue["body"]
	if present && body != nil {
		return
	}

	issue["body"] = ""
}

func issueMatchesFilters(data map[string]any, config OnIssueConfiguration) bool {
	return issueMatchesLabels(data, config.Labels, config.LabelFilterMode) &&
		issueMatchesSuperplaneLabel(data, config.SuperplaneLabelAdded) &&
		issueMatchesAssignment(data, config.Assignment)
}

func issueMatchesLabels(data map[string]any, labels []string, mode string) bool {
	labels = trimmedIssueLabels(labels)
	if len(labels) == 0 {
		return true
	}

	matched := false
	for _, name := range issueLabelNames(data) {
		if slices.Contains(labels, name) {
			matched = true
			break
		}
	}
	if mode == issueLabelFilterExclude {
		return !matched
	}
	return matched
}

func issueMatchesSuperplaneLabel(data map[string]any, enabled bool) bool {
	if !enabled {
		return true
	}
	action, _ := data["action"].(string)
	if action != "labeled" {
		return true
	}

	label, _ := data["label"].(map[string]any)
	name, _ := label["name"].(string)
	issue, _ := data["issue"].(map[string]any)
	state, _ := issue["state"].(string)
	return name == issueSuperplaneLabel && state == "open"
}

func issueMatchesAssignment(data map[string]any, assignment string) bool {
	switch assignment {
	case issueAssignmentAssigned:
		return len(issueAssignees(data)) > 0
	case issueAssignmentUnassigned:
		return len(issueAssignees(data)) == 0
	default:
		return true
	}
}

func trimmedIssueLabels(labels []string) []string {
	names := make([]string, 0, len(labels))
	for _, label := range labels {
		name := strings.TrimSpace(label)
		if name != "" {
			names = append(names, name)
		}
	}
	return names
}

func issueLabelNames(data map[string]any) []string {
	raw, _ := issueField(data, "labels").([]any)
	names := make([]string, 0, len(raw))
	for _, item := range raw {
		label, ok := item.(map[string]any)
		if !ok {
			continue
		}
		name, _ := label["name"].(string)
		name = strings.TrimSpace(name)
		if name != "" {
			names = append(names, name)
		}
	}
	return names
}

func issueAssignees(data map[string]any) []any {
	assignees, _ := issueField(data, "assignees").([]any)
	if assignees == nil {
		return []any{}
	}
	return assignees
}

func issueAuthorLogin(data map[string]any) string {
	user, _ := issueField(data, "user").(map[string]any)
	login, _ := user["login"].(string)
	return strings.TrimSpace(strings.TrimPrefix(login, "@"))
}

func issueField(data map[string]any, name string) any {
	issue, _ := data["issue"].(map[string]any)
	if issue == nil {
		return nil
	}
	return issue[name]
}

func authorHasRepositoryAccess(ctx core.WebhookRequestContext, repository, login string) (bool, int, error) {
	if login == "" {
		ctx.Logger.Info("Ignoring event - issue author login is missing")
		return false, http.StatusOK, nil
	}

	client, err := common.NewClient(ctx.Integration, ctx.HTTP)
	if err != nil {
		return false, http.StatusInternalServerError, fmt.Errorf("failed to initialize GitHub client: %w", err)
	}

	permission, _, err := client.GetRepositoryPermissionLevel(context.Background(), repository, login)
	if err != nil {
		return false, http.StatusInternalServerError, fmt.Errorf("failed to get repository permission: %w", err)
	}

	level := ""
	if permission != nil {
		level = permission.GetPermission()
	}
	if level == issuePermissionNone {
		ctx.Logger.Infof("Ignoring event - author %q has no repository access", login)
		return false, http.StatusOK, nil
	}
	return true, http.StatusOK, nil
}
