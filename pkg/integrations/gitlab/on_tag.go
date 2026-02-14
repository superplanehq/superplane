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

type OnTag struct{}

type OnTagConfiguration struct {
	Project string                    `json:"project" mapstructure:"project"`
	Tags    []configuration.Predicate `json:"tags" mapstructure:"tags"`
}

func (t *OnTag) Name() string {
	return "gitlab.onTag"
}

func (t *OnTag) Label() string {
	return "On Tag"
}

func (t *OnTag) Description() string {
	return "Listen to tag events from GitLab"
}

func (t *OnTag) Documentation() string {
	return `The On Tag trigger starts a workflow execution when tag push events occur in a GitLab project.

## Configuration

- **Project** (required): GitLab project to monitor
- **Tags** (required): Configure tag filters using predicates. You can match full refs (refs/tags/v1.0.0) or tag names (v1.0.0).

## Outputs

- **Default channel**: Emits tag push payload data including ref, before/after SHA, and project information`
}

func (t *OnTag) Icon() string {
	return "gitlab"
}

func (t *OnTag) Color() string {
	return "orange"
}

func (t *OnTag) Configuration() []configuration.Field {
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
			Name:     "tags",
			Label:    "Tags",
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

func (t *OnTag) Setup(ctx core.TriggerContext) error {
	var config OnTagConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if err := ensureProjectInMetadata(ctx.Metadata, ctx.Integration, config.Project); err != nil {
		return err
	}

	return ctx.Integration.RequestWebhook(WebhookConfiguration{
		EventType: "tag_push",
		ProjectID: config.Project,
	})
}

func (t *OnTag) Actions() []core.Action {
	return []core.Action{}
}

func (t *OnTag) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnTag) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	var config OnTagConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to decode configuration: %w", err)
	}

	eventType := ctx.Headers.Get("X-Gitlab-Event")
	if eventType == "" {
		return http.StatusBadRequest, fmt.Errorf("missing X-Gitlab-Event header")
	}

	if eventType != "Tag Push Hook" {
		return http.StatusOK, nil
	}

	code, err := verifyWebhookToken(ctx)
	if err != nil {
		return code, err
	}

	data := map[string]any{}
	if err := json.Unmarshal(ctx.Body, &data); err != nil {
		return http.StatusBadRequest, fmt.Errorf("error parsing request body: %v", err)
	}

	if len(config.Tags) > 0 && !t.matchesTag(ctx.Logger, data, config.Tags) {
		return http.StatusOK, nil
	}

	if err := ctx.Events.Emit("gitlab.tag", data); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil
}

func (t *OnTag) Cleanup(ctx core.TriggerContext) error {
	return nil
}

func (t *OnTag) matchesTag(logger *log.Entry, data map[string]any, predicates []configuration.Predicate) bool {
	ref, ok := data["ref"].(string)
	if !ok {
		return false
	}

	if configuration.MatchesAnyPredicate(predicates, ref) {
		return true
	}

	tag := strings.TrimPrefix(ref, "refs/tags/")
	if tag != ref && configuration.MatchesAnyPredicate(predicates, tag) {
		return true
	}

	logger.Infof("Tag %s does not match the allowed predicates: %v", ref, predicates)
	return false
}
