package github

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnBranchCreated struct{}

type OnBranchCreatedConfiguration struct {
	Repository string                    `json:"repository"`
	Branches   []configuration.Predicate `json:"branches"`
}

func (t *OnBranchCreated) Name() string {
	return "github.onBranchCreated"
}

func (t *OnBranchCreated) Label() string {
	return "On Branch Created"
}

func (t *OnBranchCreated) Description() string {
	return "Listen to GitHub branch creation events"
}

func (t *OnBranchCreated) Documentation() string {
	return `The On Branch Created trigger starts a workflow execution when a new branch is created in a GitHub repository.

## Use Cases

- **Branch automation**: Set up environments or resources for new branches
- **Branch validation**: Validate branch naming conventions
- **Notification workflows**: Notify teams when important branches are created
- **Branch processing**: Process or configure branches automatically

## Configuration

- **Repository**: Select the GitHub repository to monitor
- **Branches**: Configure which branches to listen for using predicates (e.g., equals "main", starts with "feature-")

## Event Data

Each branch event includes:
- **ref**: The branch reference (e.g., "refs/heads/feature/new-feature")
- **ref_type**: Type of reference (branch)
- **repository**: Repository information
- **sender**: User who created the branch

## Webhook Setup

This trigger automatically sets up a GitHub webhook when configured. The webhook is managed by SuperPlane and will be cleaned up when the trigger is removed.`
}

func (t *OnBranchCreated) Icon() string {
	return "github"
}

func (t *OnBranchCreated) Color() string {
	return "gray"
}

func (t *OnBranchCreated) Configuration() []configuration.Field {
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
	err := ensureRepoInMetadata(
		ctx.Metadata,
		ctx.Integration,
		ctx.Configuration,
	)

	if err != nil {
		return err
	}

	var config OnBranchCreatedConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	return ctx.Integration.RequestWebhook(WebhookConfiguration{
		EventType:  "create",
		Repository: config.Repository,
	})
}

func (t *OnBranchCreated) Actions() []core.Action {
	return []core.Action{}
}

func (t *OnBranchCreated) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnBranchCreated) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	ctx = withWebhookLogger(ctx, t.Name())
	ctx.Logger.Infof("Received GitHub webhook")

	config := OnBranchCreatedConfiguration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		ctx.Logger.Errorf("Failed to decode configuration: %v", err)
		return http.StatusInternalServerError, nil, fmt.Errorf("failed to decode configuration: %w", err)
	}

	eventType := ctx.Headers.Get("X-GitHub-Event")
	if eventType == "" {
		return http.StatusBadRequest, nil, fmt.Errorf("missing X-GitHub-Event header")
	}

	if eventType != "create" {
		ctx.Logger.Infof("ignoring event - event type %q is not a create event", eventType)
		return http.StatusOK, nil, nil
	}

	code, err := verifySignature(ctx)
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

	//
	// Check ref_type - only process branches, not tags
	//
	refType, ok := data["ref_type"]
	if !ok {
		ctx.Logger.Errorf("Missing ref_type")
		return http.StatusBadRequest, nil, fmt.Errorf("missing ref_type")
	}

	rt, ok := refType.(string)
	if !ok {
		ctx.Logger.Errorf("Invalid ref_type")
		return http.StatusBadRequest, nil, fmt.Errorf("invalid ref_type")
	}

	if rt != "branch" {
		ctx.Logger.Infof("Ignoring event - ref_type %q is not a branch", rt)
		return http.StatusOK, nil, nil
	}

	ref, ok := data["ref"]
	if !ok {
		ctx.Logger.Errorf("Missing ref")
		return http.StatusBadRequest, nil, fmt.Errorf("missing ref")
	}

	r, ok := ref.(string)
	if !ok {
		ctx.Logger.Errorf("Invalid ref")
		return http.StatusBadRequest, nil, fmt.Errorf("invalid ref")
	}

	if !configuration.MatchesAnyPredicate(config.Branches, r) {
		ctx.Logger.Infof("Ignoring event - ref %q did not match configured filters", r)
		return http.StatusOK, nil, nil
	}

	err = ctx.Events.Emit("github.branchCreated", data)
	if err != nil {
		ctx.Logger.Errorf("Failed to emit event: %v", err)
		return http.StatusInternalServerError, nil, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil, nil
}

func (t *OnBranchCreated) Cleanup(ctx core.TriggerContext) error {
	return nil
}
