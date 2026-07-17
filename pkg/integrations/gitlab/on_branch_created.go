package gitlab

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/mitchellh/mapstructure"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnBranchCreated struct{}

type OnBranchCreatedConfiguration struct {
	Project  string                    `json:"project" mapstructure:"project"`
	Branches []configuration.Predicate `json:"branches" mapstructure:"branches"`
}

func (t *OnBranchCreated) Name() string {
	return "gitlab.onBranchCreated"
}

func (t *OnBranchCreated) Label() string {
	return "On Branch Created"
}

func (t *OnBranchCreated) Description() string {
	return "Listen to branch creation events from GitLab"
}

func (t *OnBranchCreated) Documentation() string {
	return `The On Branch Created trigger starts a workflow execution when a new branch is created in a GitLab project.

## Use Cases

- **Preview environments**: Provision a temporary environment when a feature branch is created
- **Branch conventions**: Validate branch naming and alert on anomalies
- **Release tracking**: Open tracking issues or apply protective rules when release branches are created

## Configuration

- **Project** (required): GitLab project to monitor
- **Branches** (required): Configure branch filters using predicates. You can match full refs (refs/heads/main) or branch names (main).

## Event Data

GitLab signals branch creation with a push event whose "before" SHA is all zeros. Each event includes:
- **ref**: The branch reference that was created (e.g. refs/heads/feature/new-feature)
- **before**: All-zero SHA, indicating the branch did not exist before
- **after**: The SHA the new branch points to
- **user_name/user_username**: The user who created the branch
- **project**: Project information

## Webhook Setup

This trigger automatically sets up a GitLab webhook when configured. The webhook is managed by SuperPlane and will be cleaned up when the trigger is removed.`
}

func (t *OnBranchCreated) Icon() string {
	return "gitlab"
}

func (t *OnBranchCreated) Color() string {
	return "orange"
}

func (t *OnBranchCreated) Configuration() []configuration.Field {
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
			Name:     "branches",
			Label:    "Branches",
			Type:     configuration.FieldTypeAnyPredicateList,
			Required: true,
			Default: []map[string]any{
				{
					"type":  configuration.PredicateTypeMatches,
					"value": ".*",
				},
			},
			TypeOptions: &configuration.TypeOptions{
				AnyPredicateList: &configuration.AnyPredicateListTypeOptions{
					Operators: configuration.AllPredicateOperators,
				},
			},
		},
	}
}

func (t *OnBranchCreated) Setup(ctx core.TriggerContext) error {
	var config OnBranchCreatedConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if err := ensureProjectInMetadata(ctx.Metadata, ctx.Integration, config.Project); err != nil {
		return err
	}

	return ctx.Integration.RequestWebhook(WebhookConfiguration{
		EventType: "push",
		ProjectID: config.Project,
	})
}

func (t *OnBranchCreated) Hooks() []core.Hook {
	return []core.Hook{}
}

func (t *OnBranchCreated) HandleHook(ctx core.TriggerHookContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnBranchCreated) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	var config OnBranchCreatedConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("failed to decode configuration: %w", err)
	}

	eventType := ctx.Headers.Get("X-Gitlab-Event")
	if eventType == "" {
		return http.StatusBadRequest, nil, fmt.Errorf("missing X-Gitlab-Event header")
	}

	if eventType != "Push Hook" {
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

	//
	// GitLab does not have a dedicated branch creation event. A new branch
	// arrives as a push event whose "before" SHA is all zeros. Ignore every
	// other push so this trigger only fires on branch creation.
	//
	if !isBranchCreation(data) {
		ctx.Logger.Info("Ignoring event - not a branch creation")
		return http.StatusOK, nil, nil
	}

	if len(config.Branches) > 0 && !t.matchesBranch(ctx.Logger, data, config.Branches) {
		return http.StatusOK, nil, nil
	}

	if err := ctx.Events.Emit("gitlab.branchCreated", data); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil, nil
}

func (t *OnBranchCreated) Cleanup(ctx core.TriggerContext) error {
	return nil
}

func (t *OnBranchCreated) matchesBranch(logger *log.Entry, data map[string]any, predicates []configuration.Predicate) bool {
	ref, ok := data["ref"].(string)
	if !ok {
		return false
	}

	if configuration.MatchesAnyPredicate(predicates, ref) {
		return true
	}

	branch := strings.TrimPrefix(ref, "refs/heads/")
	if branch != ref && configuration.MatchesAnyPredicate(predicates, branch) {
		return true
	}

	logger.Infof("Branch %s does not match the allowed predicates: %v", ref, predicates)
	return false
}

// isBranchCreation reports whether a GitLab push payload represents the
// creation of a new branch, signalled by an all-zero "before" SHA.
func isBranchCreation(data map[string]any) bool {
	before, ok := data["before"].(string)
	if !ok {
		return false
	}

	return before == zeroSHA
}
