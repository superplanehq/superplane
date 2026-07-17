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

type OnPush struct{}

type OnPushConfiguration struct {
	Project  string                    `json:"project" mapstructure:"project"`
	Branches []configuration.Predicate `json:"branches" mapstructure:"branches"`
}

func (p *OnPush) Name() string {
	return "gitlab.onPush"
}

func (p *OnPush) Label() string {
	return "On Push"
}

func (p *OnPush) Description() string {
	return "Listen to push events from GitLab"
}

func (p *OnPush) Documentation() string {
	return `The On Push trigger starts a workflow execution when code is pushed to an existing branch in a GitLab project. New-branch creation is handled by the dedicated On Branch Created trigger, and branch deletions are ignored.

## Use Cases

- **CI/CD automation**: Trigger builds and deployments when code is pushed to ` + "`main`" + `
- **Policy gates**: Run linting, security, or policy checks on every push
- **Notifications**: Post a summary to Slack when code lands on a branch

## Configuration

- **Project** (required): GitLab project to monitor
- **Branches** (required): Configure branch filters using predicates. You can match full refs (refs/heads/main) or branch names (main).

## Event Data

Each push event includes:
- **ref**: The branch reference that was pushed to (e.g. refs/heads/main)
- **before/after**: Commit SHAs before and after the push
- **commits**: Array of commit information, each with added/modified/removed file lists
- **user_name/user_username**: The user who pushed
- **project**: Project information

## Webhook Setup

This trigger automatically sets up a GitLab webhook when configured. The webhook is managed by SuperPlane and will be cleaned up when the trigger is removed.`
}

func (p *OnPush) Icon() string {
	return "gitlab"
}

func (p *OnPush) Color() string {
	return "orange"
}

func (p *OnPush) Configuration() []configuration.Field {
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

func (p *OnPush) Setup(ctx core.TriggerContext) error {
	var config OnPushConfiguration
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

func (p *OnPush) Hooks() []core.Hook {
	return []core.Hook{}
}

func (p *OnPush) HandleHook(ctx core.TriggerHookContext) (map[string]any, error) {
	return nil, nil
}

func (p *OnPush) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	var config OnPushConfiguration
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
	// Branch creation (all-zero "before") and deletion (all-zero "after") are
	// ref-lifecycle events, not content pushes. Creation is handled by the
	// dedicated On Branch Created trigger, and deletion carries no commits, so
	// On Push ignores both and only fires on pushes to existing branches.
	//
	if isBranchCreation(data) {
		ctx.Logger.Info("Ignoring event - branch creation (handled by On Branch Created)")
		return http.StatusOK, nil, nil
	}

	if isBranchDeletion(data) {
		ctx.Logger.Info("Ignoring event - branch deletion")
		return http.StatusOK, nil, nil
	}

	if len(config.Branches) > 0 && !p.matchesBranch(ctx.Logger, data, config.Branches) {
		return http.StatusOK, nil, nil
	}

	if err := ctx.Events.Emit("gitlab.push", data); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil, nil
}

func (p *OnPush) Cleanup(ctx core.TriggerContext) error {
	return nil
}

func (p *OnPush) matchesBranch(logger *log.Entry, data map[string]any, predicates []configuration.Predicate) bool {
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

	logger.Infof("Ref %s does not match the allowed predicates: %v", ref, predicates)
	return false
}

// isBranchDeletion reports whether a GitLab push payload represents a branch
// deletion, signalled by an all-zero "after" SHA.
func isBranchDeletion(data map[string]any) bool {
	after, ok := data["after"].(string)
	if !ok {
		return false
	}

	return after == zeroSHA
}
