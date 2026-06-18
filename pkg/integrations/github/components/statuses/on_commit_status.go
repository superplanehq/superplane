package statuses

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
)

type OnCommitStatus struct{}

type OnCommitStatusConfiguration struct {
	Repository string                    `json:"repository" mapstructure:"repository"`
	States     []string                  `json:"states" mapstructure:"states"`
	Contexts   []configuration.Predicate `json:"contexts" mapstructure:"contexts"`
	Branches   []configuration.Predicate `json:"branches" mapstructure:"branches"`
}

func (t *OnCommitStatus) Name() string {
	return "github.onCommitStatus"
}

func (t *OnCommitStatus) Label() string {
	return "On Commit Status"
}

func (t *OnCommitStatus) Description() string {
	return "Listen to GitHub commit status events from the Commit Statuses API"
}

func (t *OnCommitStatus) Documentation() string {
	return `The On Commit Status trigger starts a workflow execution when a GitHub commit status is created or updated.

GitHub commit statuses are the legacy status objects created through the Commit Statuses API. They are separate from GitHub Checks API check runs, which power many PR checks from GitHub Apps such as Cloudflare Pages, DCO, and Sourcery. This trigger does not receive those check-run events. For GitHub Actions workflows, use the On Workflow Run trigger instead.

## Use Cases

- **Merge automation**: Re-evaluate pull request merge criteria when required status checks change
- **CI/CD orchestration**: React to status updates from external CI systems
- **Quality gates**: Route workflows based on commit status context and state
- **Notifications**: Notify teams when important status checks fail or recover

## Configuration

- **Repository**: Select the GitHub repository to monitor
- **States**: Select which commit status states to listen for (error, failure, pending, success)
- **Contexts** *(optional)*: Configure which status contexts to listen for using predicates (e.g., equals "ci/build", matches "deploy/.*")
- **Branches** *(optional)*: Configure which branch names to listen for using predicates. GitHub includes branches that contain the status SHA.

## Event Data

Each status event includes:
- **state**: The new commit status state (error, failure, pending, success)
- **context**: The status context, such as "ci/build"
- **sha**: Commit SHA for the status
- **description**: Optional status description
- **target_url**: Optional link added to the status
- **branches**: Branches containing the status SHA
- **commit**: Commit information
- **repository**: Repository information
- **sender**: User who created the status event. This is not necessarily the commit author

## Webhook Setup

This trigger automatically sets up a GitHub webhook when configured. The webhook is managed by SuperPlane and will be cleaned up when the trigger is removed.`
}

func (t *OnCommitStatus) Icon() string {
	return "github"
}

func (t *OnCommitStatus) Color() string {
	return "gray"
}

func (t *OnCommitStatus) Configuration() []configuration.Field {
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
			Name:     "states",
			Label:    "States",
			Type:     configuration.FieldTypeMultiSelect,
			Required: true,
			Default:  []string{"error", "failure", "pending", "success"},
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Error", Value: "error"},
						{Label: "Failure", Value: "failure"},
						{Label: "Pending", Value: "pending"},
						{Label: "Success", Value: "success"},
					},
				},
			},
		},
		{
			Name:        "contexts",
			Label:       "Contexts",
			Description: "Optional. Filter commit status contexts, e.g. ci/build or deploy/production.",
			Type:        configuration.FieldTypeAnyPredicateList,
			Required:    false,
			Togglable:   true,
			TypeOptions: &configuration.TypeOptions{
				AnyPredicateList: &configuration.AnyPredicateListTypeOptions{
					Operators: configuration.AllPredicateOperators,
				},
			},
		},
		{
			Name:        "branches",
			Label:       "Branches",
			Description: "Optional. Filter branch names that contain the status SHA.",
			Type:        configuration.FieldTypeAnyPredicateList,
			Required:    false,
			Togglable:   true,
			TypeOptions: &configuration.TypeOptions{
				AnyPredicateList: &configuration.AnyPredicateListTypeOptions{
					Operators: configuration.AllPredicateOperators,
				},
			},
		},
	}
}

func (t *OnCommitStatus) Setup(ctx core.TriggerContext) error {
	err := common.EnsureRepoInMetadata(
		ctx.Metadata,
		ctx.Integration,
		ctx.HTTP,
		ctx.Configuration,
	)

	if err != nil {
		return err
	}

	var config OnCommitStatusConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	return ctx.Integration.RequestWebhook(common.WebhookConfiguration{
		EventType:  "status",
		Repository: config.Repository,
	})
}

func (t *OnCommitStatus) Hooks() []core.Hook {
	return []core.Hook{}
}

func (t *OnCommitStatus) HandleHook(ctx core.TriggerHookContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnCommitStatus) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	ctx = common.WithWebhookLogger(ctx, t.Name())
	ctx.Logger.Infof("Received GitHub webhook")

	config := OnCommitStatusConfiguration{}
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

	if eventType != "status" {
		ctx.Logger.Infof("Ignoring event - event type %q is not a status event", eventType)
		return http.StatusOK, nil, nil
	}

	code, err := common.VerifySignature(ctx)
	if err != nil {
		ctx.Logger.Errorf("Failed to verify signature: %v", err)
		return code, nil, err
	}

	payload := map[string]any{}
	err = json.Unmarshal(ctx.Body, &payload)
	if err != nil {
		ctx.Logger.Errorf("Failed to parse request body: %v", err)
		return http.StatusBadRequest, nil, fmt.Errorf("error parsing request body: %v", err)
	}

	statusState, ok := extractStatusState(payload)
	if !ok {
		ctx.Logger.Errorf("Missing or invalid status state")
		return http.StatusBadRequest, nil, fmt.Errorf("missing or invalid status state")
	}

	if !matchesConfiguredStatusState(statusState, config.States) {
		ctx.Logger.Infof("Ignoring event - state %q did not match configured filters", statusState)
		return http.StatusOK, nil, nil
	}

	statusContext, ok := extractStatusContext(payload)
	if !ok {
		ctx.Logger.Errorf("Missing or invalid status context")
		return http.StatusBadRequest, nil, fmt.Errorf("missing or invalid status context")
	}

	if !matchesStatusContext(statusContext, config.Contexts) {
		ctx.Logger.Infof("Ignoring event - context %q did not match configured filters", statusContext)
		return http.StatusOK, nil, nil
	}

	if !matchesStatusBranches(payload, config.Branches) {
		ctx.Logger.Infof("Ignoring event - branches did not match configured filters")
		return http.StatusOK, nil, nil
	}

	err = ctx.Events.Emit("github.status", payload)
	if err != nil {
		ctx.Logger.Errorf("Failed to emit event: %v", err)
		return http.StatusInternalServerError, nil, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil, nil
}

func matchesConfiguredStatusState(statusState string, allowedStates []string) bool {
	if len(allowedStates) == 0 {
		return true
	}

	return slices.Contains(allowedStates, statusState)
}

func matchesStatusContext(statusContext string, allowedContexts []configuration.Predicate) bool {
	if len(allowedContexts) == 0 {
		return true
	}

	return configuration.MatchesAnyPredicate(allowedContexts, statusContext)
}

func matchesStatusBranches(payload map[string]any, allowedBranches []configuration.Predicate) bool {
	if len(allowedBranches) == 0 {
		return true
	}

	return configuration.MatchesAnyPredicateInList(allowedBranches, extractStatusBranches(payload))
}

func extractStatusState(payload map[string]any) (string, bool) {
	return extractString(payload, "state")
}

func extractStatusContext(payload map[string]any) (string, bool) {
	return extractString(payload, "context")
}

func extractString(payload map[string]any, key string) (string, bool) {
	value, ok := payload[key]
	if !ok {
		return "", false
	}

	text, ok := value.(string)
	if !ok {
		return "", false
	}

	return text, true
}

func extractStatusBranches(payload map[string]any) []string {
	branchesRaw, ok := payload["branches"]
	if !ok {
		return nil
	}

	branches, ok := branchesRaw.([]any)
	if !ok {
		return nil
	}

	names := []string{}
	for _, branchRaw := range branches {
		branch, ok := branchRaw.(map[string]any)
		if !ok {
			continue
		}

		name, ok := branch["name"].(string)
		if ok {
			names = append(names, name)
		}
	}

	return names
}

func (t *OnCommitStatus) Cleanup(ctx core.TriggerContext) error {
	return nil
}
