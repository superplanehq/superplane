package ecr

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnImagePush struct{}

type OnImagePushConfiguration struct {
	Repositories []configuration.Predicate `json:"repositories" mapstructure:"repositories"`
	ImageTags    []configuration.Predicate `json:"imageTags" mapstructure:"imageTags"`
}

func (p *OnImagePush) Name() string {
	return "aws.ecr.onImagePush"
}

func (p *OnImagePush) Label() string {
	return "ECR - On Image Push"
}

func (p *OnImagePush) Description() string {
	return "Listen to AWS ECR image push events"
}

func (p *OnImagePush) Documentation() string {
	return `The On Image Push trigger starts a workflow execution when an image is pushed to an ECR repository.

## Use Cases

- **Build pipelines**: Trigger builds and deployments on container pushes
- **Security automation**: Kick off scans or alerts for newly pushed images
- **Release workflows**: Promote artifacts when a tag is published

## Configuration

- **Repositories**: Optional filters for ECR repository names
- **Image Tags**: Optional filters for image tags (for example: ` + "`latest`" + ` or ` + "`^v[0-9]+`" + `)

## Event Data

Each image push event includes:
- **detail.repository-name**: ECR repository name
- **detail.image-tag**: Tag that was pushed
- **detail.image-digest**: Digest of the image

## EventBridge Setup

This trigger automatically creates an EventBridge API destination and rule so AWS can deliver ECR events to SuperPlane.`
}

func (p *OnImagePush) Icon() string {
	return "aws"
}

func (p *OnImagePush) Color() string {
	return "orange"
}

func (p *OnImagePush) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "repositories",
			Label:       "Repositories",
			Type:        configuration.FieldTypeAnyPredicateList,
			Required:    false,
			Description: "Filter by ECR repository name",
			TypeOptions: &configuration.TypeOptions{
				AnyPredicateList: &configuration.AnyPredicateListTypeOptions{
					Operators: configuration.AllPredicateOperators,
				},
			},
		},
		{
			Name:        "imageTags",
			Label:       "Image Tags",
			Type:        configuration.FieldTypeAnyPredicateList,
			Required:    false,
			Description: "Filter by ECR image tag",
			TypeOptions: &configuration.TypeOptions{
				AnyPredicateList: &configuration.AnyPredicateListTypeOptions{
					Operators: configuration.AllPredicateOperators,
				},
			},
		},
	}
}

func (p *OnImagePush) Setup(ctx core.TriggerContext) error {
	return ctx.Integration.RequestWebhook(WebhookConfiguration{
		EventType: WebhookEventImagePush,
	})
}

func (p *OnImagePush) Actions() []core.Action {
	return []core.Action{}
}

func (p *OnImagePush) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (p *OnImagePush) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	config := OnImagePushConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to decode configuration: %w", err)
	}

	apiKey := ctx.Headers.Get(apiKeyHeaderName)
	if apiKey == "" {
		return http.StatusForbidden, fmt.Errorf("missing webhook signature")
	}

	secret, err := ctx.Webhook.GetSecret()
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error authenticating request")
	}

	if subtle.ConstantTimeCompare([]byte(apiKey), secret) != 1 {
		return http.StatusForbidden, fmt.Errorf("invalid webhook signature")
	}

	data := map[string]any{}
	if err := json.Unmarshal(ctx.Body, &data); err != nil {
		return http.StatusBadRequest, fmt.Errorf("error parsing request body: %v", err)
	}

	if !isImagePushEvent(data) {
		return http.StatusOK, nil
	}

	repositoryName := nestedString(data, "detail", "repository-name")
	imageTag := nestedString(data, "detail", "image-tag")

	if !matchesPredicates(config.Repositories, repositoryName) {
		return http.StatusOK, nil
	}

	if !matchesPredicates(config.ImageTags, imageTag) {
		return http.StatusOK, nil
	}

	if err := ctx.Events.Emit("aws.ecr.image.push", data); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil
}

func isImagePushEvent(data map[string]any) bool {
	actionType := nestedString(data, "detail", "action-type")
	if actionType != "PUSH" {
		return false
	}

	result := nestedString(data, "detail", "result")
	if result == "" {
		return true
	}

	return result == "SUCCESS"
}

func nestedString(data map[string]any, keys ...string) string {
	var current any = data
	for _, key := range keys {
		next, ok := current.(map[string]any)
		if !ok {
			return ""
		}

		current, ok = next[key]
		if !ok {
			return ""
		}
	}

	value, _ := current.(string)
	return value
}

func matchesPredicates(predicates []configuration.Predicate, value string) bool {
	if len(predicates) == 0 {
		return true
	}

	if value == "" {
		return false
	}

	return configuration.MatchesAnyPredicate(predicates, value)
}
