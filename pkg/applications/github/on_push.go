package github

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnPush struct{}

type OnPushConfiguration struct {
	Repository string                    `json:"repository" mapstructure:"repository"`
	Refs       []configuration.Predicate `json:"refs" mapstructure:"refs"`
}

func (p *OnPush) Name() string {
	return "github.onPush"
}

func (p *OnPush) Label() string {
	return "On Push"
}

func (p *OnPush) Description() string {
	return "Listen to GitHub push events"
}

func (p *OnPush) Icon() string {
	return "github"
}

func (p *OnPush) Color() string {
	return "gray"
}

func (p *OnPush) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "repository",
			Label:    "Repository",
			Type:     configuration.FieldTypeString,
			Required: true,
		},
		{
			Name:     "refs",
			Label:    "Refs",
			Type:     configuration.FieldTypeAnyPredicateList,
			Required: true,
			Default: []map[string]any{
				{
					"type":  configuration.PredicateTypeEquals,
					"value": "refs/heads/main",
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
	err := ensureRepoInMetadata(
		ctx.MetadataContext,
		ctx.AppInstallationContext,
		ctx.Configuration,
	)

	if err != nil {
		return err
	}

	var config OnPushConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	return ctx.AppInstallationContext.RequestWebhook(WebhookConfiguration{
		EventType:  "push",
		Repository: config.Repository,
	})
}

func (p *OnPush) Actions() []core.Action {
	return []core.Action{}
}

func (p *OnPush) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (p *OnPush) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	config := OnPushConfiguration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to decode configuration: %w", err)
	}

	code, err := verifySignature(ctx, "push")
	if err != nil {
		return code, err
	}

	data := map[string]any{}
	err = json.Unmarshal(ctx.Body, &data)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("error parsing request body: %v", err)
	}

	//
	// If the event is a push event for branch deletion, ignore it.
	//
	if isBranchDeletionEvent(data) {
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

	if !configuration.MatchesAnyPredicate(config.Refs, r) {
		return http.StatusOK, nil
	}

	err = ctx.EventContext.Emit("github.push", data)

	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil
}

func isBranchDeletionEvent(data map[string]any) bool {
	v, ok := data["deleted"]
	if !ok {
		return false
	}

	deleted, ok := v.(bool)
	if !ok {
		return false
	}

	return deleted
}
