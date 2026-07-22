package gitlab

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"

	"github.com/mitchellh/mapstructure"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnMergeRequest struct{}

type OnMergeRequestConfiguration struct {
	Project string   `json:"project" mapstructure:"project"`
	Actions []string `json:"actions" mapstructure:"actions"`
}

func (m *OnMergeRequest) Name() string {
	return "gitlab.onMergeRequest"
}

func (m *OnMergeRequest) Label() string {
	return "On Merge Request"
}

func (m *OnMergeRequest) Description() string {
	return "Listen to merge request events from GitLab"
}

func (m *OnMergeRequest) Documentation() string {
	return `The On Merge Request trigger starts a workflow execution when merge request events occur in a GitLab project.

## Use Cases

- **MR automation**: Automate actions when merge requests are opened, merged, or closed
- **Code review workflows**: Trigger review processes or notifications
- **CI/CD integration**: Run tests, builds, or preview environments on merge request events
- **Status updates**: Update systems when merge request status changes

## Configuration

- **Project** (required): GitLab project to monitor
- **Actions** (required): Select which merge request actions to listen for (open, close, reopen, update, approved, merge, etc.). Default: open.

## Actions

GitLab reports most merge request changes (labels, assignees, milestone, draft toggling, reviewers, auto-merge, new commits pushed) under a single ` + "`update`" + ` action. Labeled, unlabeled, assigned, unassigned, milestoned, demilestoned, synchronize, ready for review, converted to draft, review requested, review request removed, auto merge enabled/disabled and edited are derived from the ` + "`object_attributes`/`changes`" + ` fields of an update event, to match the granularity of GitHub's equivalent trigger.

## Event Data

Each merge request event includes:
- **object_attributes**: Complete merge request information including title, description, state, action, source/target branches, and URL
- **changes**: When the merge request is updated, includes what changed (title, description, labels, etc.)
- **assignees**: Users assigned to the merge request
- **reviewers**: Users requested to review the merge request
- **labels**: Labels applied to the merge request
- **project**: Project information
- **repository**: Repository information
- **user**: User who triggered the event

Common expression paths:
- Merge request IID: ` + "`root().data.object_attributes.iid`" + `
- Merge request title: ` + "`root().data.object_attributes.title`" + `
- Action: ` + "`root().data.object_attributes.action`" + `
- State: ` + "`root().data.object_attributes.state`" + `
- Source branch: ` + "`root().data.object_attributes.source_branch`" + `
- Target branch: ` + "`root().data.object_attributes.target_branch`" + `
- Merge request URL: ` + "`root().data.object_attributes.url`" + `

## Webhook Setup

This trigger automatically sets up a GitLab webhook when configured. The webhook is managed by SuperPlane and will be cleaned up when the trigger is removed.`
}

func (m *OnMergeRequest) Icon() string {
	return "gitlab"
}

func (m *OnMergeRequest) Color() string {
	return "orange"
}

func (m *OnMergeRequest) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "project",
			Label:    "Project",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeProject,
				},
			},
		},
		{
			Name:     "actions",
			Label:    "Actions",
			Type:     configuration.FieldTypeMultiSelect,
			Required: true,
			Default:  []string{"open"},
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Opened", Value: "open"},
						{Label: "Closed", Value: "close"},
						{Label: "Reopened", Value: "reopen"},
						{Label: "Updated", Value: "update"},
						{Label: "Approval Added", Value: "approval"},
						{Label: "Approved", Value: "approved"},
						{Label: "Approval Removed", Value: "unapproval"},
						{Label: "Unapproved", Value: "unapproved"},
						{Label: "Merged", Value: "merge"},
						{Label: "Edited", Value: "edited"},
						{Label: "Assigned", Value: "assigned"},
						{Label: "Unassigned", Value: "unassigned"},
						{Label: "Labeled", Value: "labeled"},
						{Label: "Unlabeled", Value: "unlabeled"},
						{Label: "Synchronize", Value: "synchronize"},
						{Label: "Milestoned", Value: "milestoned"},
						{Label: "Demilestoned", Value: "demilestoned"},
						{Label: "Ready for review", Value: "ready_for_review"},
						{Label: "Converted to draft", Value: "converted_to_draft"},
						{Label: "Review requested", Value: "review_requested"},
						{Label: "Review request removed", Value: "review_request_removed"},
						{Label: "Auto merge enabled", Value: "auto_merge_enabled"},
						{Label: "Auto merge disabled", Value: "auto_merge_disabled"},
					},
				},
			},
		},
	}
}

func (m *OnMergeRequest) Setup(ctx core.TriggerContext) error {
	var config OnMergeRequestConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if err := ensureProjectInMetadata(ctx.Metadata, ctx.Integration, config.Project); err != nil {
		return err
	}

	return ctx.Integration.RequestWebhook(WebhookConfiguration{
		EventType: "merge_requests",
		ProjectID: config.Project,
	})
}

func (m *OnMergeRequest) Hooks() []core.Hook {
	return []core.Hook{}
}

func (m *OnMergeRequest) HandleHook(ctx core.TriggerHookContext) (map[string]any, error) {
	return nil, nil
}

func (m *OnMergeRequest) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	var config OnMergeRequestConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("failed to decode configuration: %w", err)
	}

	eventType := ctx.Headers.Get("X-Gitlab-Event")
	if eventType == "" {
		return http.StatusBadRequest, nil, fmt.Errorf("missing X-Gitlab-Event header")
	}

	if eventType != "Merge Request Hook" {
		return http.StatusOK, nil, nil
	}

	code, err := verifyWebhookToken(ctx)
	if err != nil {
		return code, nil, err
	}

	data := map[string]any{}
	if err := json.Unmarshal(ctx.Body, &data); err != nil {
		return http.StatusBadRequest, nil, fmt.Errorf("error parsing request body: %v", err)
	}

	if len(config.Actions) > 0 && !m.whitelistedAction(ctx.Logger, data, config.Actions) {
		return http.StatusOK, nil, nil
	}

	if err := ctx.Events.Emit("gitlab.mergeRequest", data); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil, nil
}

func (m *OnMergeRequest) Cleanup(ctx core.TriggerContext) error {
	return nil
}

func (m *OnMergeRequest) whitelistedAction(logger *log.Entry, data map[string]any, allowedActions []string) bool {
	attrs, ok := data["object_attributes"].(map[string]any)
	if !ok {
		return false
	}

	action, ok := attrs["action"].(string)
	if !ok {
		return false
	}

	if slices.Contains(allowedActions, action) {
		return true
	}

	// GitLab reports most merge request changes as a single "update" action; derive the finer-grained ones below.
	if action != "update" {
		logger.Infof("Action %s is not in the allowed list: %v", action, allowedActions)
		return false
	}

	derived := mergeRequestDerivedActions(attrs, data["changes"])
	for _, derivedAction := range derived {
		if slices.Contains(allowedActions, derivedAction) {
			return true
		}
	}

	logger.Infof("Update action (derived: %v) is not in the allowed list: %v", derived, allowedActions)
	return false
}

// mergeRequestDerivedActions derives GitHub-equivalent actions from an update event's object_attributes/changes.
func mergeRequestDerivedActions(attrs map[string]any, rawChanges any) []string {
	var derived []string

	if oldRev, ok := attrs["oldrev"].(string); ok && oldRev != "" {
		derived = append(derived, "synchronize")
	}

	changes, ok := rawChanges.(map[string]any)
	if !ok {
		return derived
	}

	if listGrew(changes, "labels", "id") {
		derived = append(derived, "labeled")
	}
	if listShrank(changes, "labels", "id") {
		derived = append(derived, "unlabeled")
	}
	if listGrew(changes, "assignees", "id") {
		derived = append(derived, "assigned")
	}
	if listShrank(changes, "assignees", "id") {
		derived = append(derived, "unassigned")
	}
	if listGrew(changes, "reviewers", "id") {
		derived = append(derived, "review_requested")
	}
	if listShrank(changes, "reviewers", "id") {
		derived = append(derived, "review_request_removed")
	}
	if changedToValue(changes, "milestone_id") {
		derived = append(derived, "milestoned")
	}
	if changedToNil(changes, "milestone_id") {
		derived = append(derived, "demilestoned")
	}
	if changedBoolTo(changes, "draft", false) {
		derived = append(derived, "ready_for_review")
	}
	if changedBoolTo(changes, "draft", true) {
		derived = append(derived, "converted_to_draft")
	}
	if changedBoolTo(changes, "merge_when_pipeline_succeeds", true) {
		derived = append(derived, "auto_merge_enabled")
	}
	if changedBoolTo(changes, "merge_when_pipeline_succeeds", false) {
		derived = append(derived, "auto_merge_disabled")
	}
	if changedField(changes, "title") || changedField(changes, "description") {
		derived = append(derived, "edited")
	}

	return derived
}
