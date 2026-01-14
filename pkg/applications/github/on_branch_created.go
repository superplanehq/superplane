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
			Type:     configuration.FieldTypeAppInstallationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "repository",
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
		ctx.AppInstallation,
		ctx.Configuration,
	)

	if err != nil {
		return err
	}

	var config OnBranchCreatedConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	return ctx.AppInstallation.RequestWebhook(WebhookConfiguration{
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

func (t *OnBranchCreated) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	config := OnBranchCreatedConfiguration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to decode configuration: %w", err)
	}

	code, err := verifySignature(ctx, "create")
	if err != nil {
		return code, err
	}

	data := map[string]any{}
	err = json.Unmarshal(ctx.Body, &data)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("error parsing request body: %v", err)
	}

	//
	// Check ref_type - only process branches, not tags
	//
	refType, ok := data["ref_type"]
	if !ok {
		return http.StatusBadRequest, fmt.Errorf("missing ref_type")
	}

	rt, ok := refType.(string)
	if !ok {
		return http.StatusBadRequest, fmt.Errorf("invalid ref_type")
	}

	if rt != "branch" {
		return http.StatusOK, nil
	}

	ref, ok := data["ref"]
	if !ok {
		return http.StatusBadRequest, fmt.Errorf("missing ref")
	}

	r, ok := ref.(string)
	if !ok {
		return http.StatusBadRequest, fmt.Errorf("invalid ref")
	}

	if !configuration.MatchesAnyPredicate(config.Branches, r) {
		return http.StatusOK, nil
	}

	err = ctx.Events.Emit("github.branchCreated", data)

	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil
}
