package bitbucket

import (
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
)

type OnPush struct{}

type OnPushConfiguration struct {
	Repository string                    `json:"repository" mapstructure:"repository"`
	Refs       []configuration.Predicate `json:"refs" mapstructure:"refs"`
}

func (p *OnPush) Name() string {
	return "bitbucket.onPush"
}

func (p *OnPush) Label() string {
	return "On Push"
}

func (p *OnPush) Description() string {
	return "Listen to Bitbucket push events"
}

func (p *OnPush) Documentation() string {
	return `The On Push trigger starts a workflow execution when code is pushed to a Bitbucket repository.

## Use Cases

- **CI/CD automation**: Trigger builds and deployments on code pushes
- **Code quality checks**: Run linting and tests on every push
- **Notification workflows**: Send notifications when code is pushed

## Configuration

- **Repository**: Select the Bitbucket repository to monitor
- **Refs**: Configure which branches to monitor (e.g., ` + "`refs/heads/main`" + `)

## Event Data

Each push event includes:
- **repository**: Repository information
- **push.changes**: Array of reference changes with new/old commit details
- **actor**: Information about who pushed

A single push can update more than one ref at once (for example ` + "`git push --all`" + `, or
pushing a branch and a tag together). One event is emitted per updated ref that matches
the configured refs, and ` + "`push.changes`" + ` holds only the ref that triggered the event.

## Webhook Setup

This trigger automatically sets up a Bitbucket webhook when configured. The webhook is managed by SuperPlane and will be cleaned up when the trigger is removed.`
}

func (p *OnPush) Icon() string {
	return "bitbucket"
}

func (p *OnPush) Color() string {
	return "blue"
}

func (p *OnPush) Configuration() []configuration.Field {
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
	config := OnPushConfiguration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	repo, err := ensureRepoInMetadata(ctx.HTTP, ctx.Metadata, ctx.Integration, config.Repository)
	if err != nil {
		return err
	}

	return ctx.Integration.RequestWebhook(WebhookConfiguration{
		EventTypes:     []string{"repo:push"},
		RepositorySlug: repo.Slug,
	})
}

func (p *OnPush) Hooks() []core.Hook {
	return []core.Hook{}
}

func (p *OnPush) HandleHook(ctx core.TriggerHookContext) (map[string]any, error) {
	return nil, nil
}

func (p *OnPush) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	//
	// Verify the event type.
	//
	eventKey := ctx.Headers.Get("X-Event-Key")
	if eventKey == "" {
		return http.StatusBadRequest, nil, fmt.Errorf("missing X-Event-Key header")
	}

	if eventKey != "repo:push" {
		return http.StatusOK, nil, nil
	}

	//
	// Verify the webhook signature.
	//
	signature := ctx.Headers.Get("X-Hub-Signature")
	if signature == "" {
		return http.StatusForbidden, nil, fmt.Errorf("missing X-Hub-Signature header")
	}

	signature = strings.TrimPrefix(signature, "sha256=")
	if signature == "" {
		return http.StatusForbidden, nil, fmt.Errorf("invalid signature format")
	}

	secret, err := ctx.Webhook.GetSecret()
	if err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("error getting webhook secret")
	}

	if err := crypto.VerifySignature(secret, ctx.Body, signature); err != nil {
		return http.StatusForbidden, nil, fmt.Errorf("invalid signature")
	}

	//
	// Parse the webhook payload.
	//
	data := map[string]any{}
	err = json.Unmarshal(ctx.Body, &data)
	if err != nil {
		return http.StatusBadRequest, nil, fmt.Errorf("error parsing request body: %v", err)
	}

	//
	// A single push can update several refs at once - for example
	// `git push --all`, or pushing a branch and a tag together.
	// Bitbucket reports each one as an entry in `push.changes`,
	// so every change is filtered and emitted on its own.
	//
	config := OnPushConfiguration{}
	err = mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("failed to decode configuration: %w", err)
	}

	for _, change := range extractChanges(data) {
		ref := extractRef(change)
		if ref == "" {
			continue
		}

		if !configuration.MatchesAnyPredicate(config.Refs, ref) {
			continue
		}

		err = ctx.Events.Emit("bitbucket.push", payloadForChange(data, change))
		if err != nil {
			return http.StatusInternalServerError, nil, fmt.Errorf("error emitting event: %v", err)
		}
	}

	return http.StatusOK, nil, nil
}

func (p *OnPush) Cleanup(ctx core.TriggerContext) error {
	return nil
}

// extractChanges returns the ref changes carried by a Bitbucket push payload.
func extractChanges(data map[string]any) []map[string]any {
	push, ok := data["push"].(map[string]any)
	if !ok {
		return nil
	}

	rawChanges, ok := push["changes"].([]any)
	if !ok {
		return nil
	}

	changes := make([]map[string]any, 0, len(rawChanges))
	for _, rawChange := range rawChanges {
		change, ok := rawChange.(map[string]any)
		if !ok {
			continue
		}

		changes = append(changes, change)
	}

	return changes
}

// payloadForChange returns the push payload narrowed down to a single ref change,
// so that `push.changes` always describes the ref that triggered the event.
func payloadForChange(data map[string]any, change map[string]any) map[string]any {
	push, ok := data["push"].(map[string]any)
	if !ok {
		return data
	}

	narrowedPush := maps.Clone(push)
	narrowedPush["changes"] = []any{change}

	payload := maps.Clone(data)
	payload["push"] = narrowedPush

	return payload
}

// extractRef extracts the ref name from a single Bitbucket push change.
func extractRef(change map[string]any) string {
	newRef, ok := change["new"].(map[string]any)
	if !ok {
		return ""
	}

	refType, _ := newRef["type"].(string)
	refName, _ := newRef["name"].(string)

	if refName == "" {
		return ""
	}

	switch refType {
	case "branch":
		return fmt.Sprintf("refs/heads/%s", refName)
	case "tag":
		return fmt.Sprintf("refs/tags/%s", refName)
	default:
		return fmt.Sprintf("refs/heads/%s", refName)
	}
}
