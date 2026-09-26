package bitbucket

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

// Bitbucket Pipelines — and any external CI reporting through the build status API —
// surface their result as a commit status, so this is the trigger that tells a canvas
// a build finished.
var commitStatusEventTypes = []string{"repo:commit_status_created", "repo:commit_status_updated"}

var commitStatusStates = []configuration.FieldOption{
	{Label: "In progress", Value: StateInProgress},
	{Label: "Successful", Value: StateSuccessful},
	{Label: "Failed", Value: StateFailed},
	{Label: "Stopped", Value: StateStopped},
}

type OnCommitStatus struct{}

type OnCommitStatusConfiguration struct {
	Repository string                    `json:"repository" mapstructure:"repository"`
	States     []string                  `json:"states" mapstructure:"states"`
	Keys       []configuration.Predicate `json:"keys" mapstructure:"keys"`
	Refs       []configuration.Predicate `json:"refs" mapstructure:"refs"`
}

func (s *OnCommitStatus) Name() string {
	return "bitbucket.onCommitStatus"
}

func (s *OnCommitStatus) Label() string {
	return "On Commit Status"
}

func (s *OnCommitStatus) Description() string {
	return "Listen to Bitbucket build statuses, including Bitbucket Pipelines results"
}

func (s *OnCommitStatus) Documentation() string {
	return `The On Commit Status trigger starts a workflow execution when a build status is reported on a commit.

Bitbucket Pipelines reports every pipeline result through the build status API, so this trigger is how a
canvas reacts to a pipeline finishing. External CI systems that publish statuses to Bitbucket — including
SuperPlane's own **Publish Commit Status** component — fire this trigger too.

## Use Cases

- **Deploy on green**: Start a deployment when the build for a commit reports ` + "`SUCCESSFUL`" + `
- **Fail fast**: Open an incident or notify a channel when a build reports ` + "`FAILED`" + `
- **Release gating**: Wait for a specific build key, such as ` + "`integration-tests`" + `, before promoting a build

## Configuration

- **Repository** (required): The Bitbucket repository to monitor
- **States** (required): Which build states should start a run. Default: Successful and Failed.
- **Keys** (optional): Only fire for statuses whose key matches, e.g. ` + "`integration-tests`" + `. Leave empty to accept every key.
- **Refs** (optional): Only fire for statuses reported on a matching branch or tag. Leave empty to accept any ref.

## Event Data

Each event includes:
- **commit_status**: The status, including ` + "`key`" + `, ` + "`name`" + `, ` + "`state`" + `, ` + "`url`" + `, ` + "`refname`" + ` and ` + "`links`" + `
- **repository**: Repository information
- **actor**: The user or app that reported the status

Common expression paths:
- Build state: ` + "`root().data.commit_status.state`" + `
- Build key: ` + "`root().data.commit_status.key`" + `
- Build URL: ` + "`root().data.commit_status.url`" + `
- Branch: ` + "`root().data.commit_status.refname`" + `

The commit the status belongs to is referenced by ` + "`root().data.commit_status.links.commit.href`" + `; its
last path segment is the commit hash.

## Webhook Setup

This trigger automatically sets up a Bitbucket webhook when configured. The webhook is managed by SuperPlane and will be cleaned up when the trigger is removed.`
}

func (s *OnCommitStatus) Icon() string {
	return "bitbucket"
}

func (s *OnCommitStatus) Color() string {
	return "blue"
}

func (s *OnCommitStatus) Configuration() []configuration.Field {
	return []configuration.Field{
		repositoryField(),
		{
			Name:        "states",
			Label:       "States",
			Type:        configuration.FieldTypeMultiSelect,
			Required:    true,
			Default:     []string{StateSuccessful, StateFailed},
			Description: "Which build states should start a run",
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: commitStatusStates,
				},
			},
		},
		{
			Name:        "keys",
			Label:       "Keys",
			Type:        configuration.FieldTypeAnyPredicateList,
			Required:    false,
			Description: "Only fire for build statuses whose key matches. Leave empty to accept any key.",
			TypeOptions: &configuration.TypeOptions{
				AnyPredicateList: &configuration.AnyPredicateListTypeOptions{
					Operators: configuration.AllPredicateOperators,
				},
			},
		},
		{
			Name:        "refs",
			Label:       "Refs",
			Type:        configuration.FieldTypeAnyPredicateList,
			Required:    false,
			Description: "Only fire for build statuses reported on a matching branch or tag. Leave empty to accept any ref.",
			TypeOptions: &configuration.TypeOptions{
				AnyPredicateList: &configuration.AnyPredicateListTypeOptions{
					Operators: configuration.AllPredicateOperators,
				},
			},
		},
	}
}

func (s *OnCommitStatus) Setup(ctx core.TriggerContext) error {
	config := OnCommitStatusConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if len(config.States) == 0 {
		return fmt.Errorf("at least one state is required")
	}

	for _, state := range config.States {
		known := slices.ContainsFunc(commitStatusStates, func(option configuration.FieldOption) bool {
			return option.Value == state
		})

		if !known {
			return fmt.Errorf("unsupported build state %q", state)
		}
	}

	repo, err := ensureRepoInMetadata(ctx.HTTP, ctx.Metadata, ctx.Integration, config.Repository)
	if err != nil {
		return err
	}

	return ctx.Integration.RequestWebhook(WebhookConfiguration{
		EventTypes:     commitStatusEventTypes,
		RepositorySlug: repo.Slug,
	})
}

func (s *OnCommitStatus) Hooks() []core.Hook {
	return []core.Hook{}
}

func (s *OnCommitStatus) HandleHook(ctx core.TriggerHookContext) (map[string]any, error) {
	return nil, nil
}

func (s *OnCommitStatus) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	config := OnCommitStatusConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("failed to decode configuration: %w", err)
	}

	eventKey := ctx.Headers.Get("X-Event-Key")
	if eventKey == "" {
		return http.StatusBadRequest, nil, fmt.Errorf("missing X-Event-Key header")
	}

	if !slices.Contains(commitStatusEventTypes, eventKey) {
		return http.StatusOK, nil, nil
	}

	if code, err := verifyWebhookSignature(ctx); err != nil {
		return code, nil, err
	}

	data := map[string]any{}
	if err := json.Unmarshal(ctx.Body, &data); err != nil {
		return http.StatusBadRequest, nil, fmt.Errorf("error parsing request body: %v", err)
	}

	status, ok := data["commit_status"].(map[string]any)
	if !ok {
		return http.StatusOK, nil, nil
	}

	state, _ := status["state"].(string)
	if !slices.Contains(config.States, strings.ToUpper(strings.TrimSpace(state))) {
		return http.StatusOK, nil, nil
	}

	if len(config.Keys) > 0 {
		key, _ := status["key"].(string)
		if !configuration.MatchesAnyPredicate(config.Keys, key) {
			return http.StatusOK, nil, nil
		}
	}

	if len(config.Refs) > 0 {
		refname, _ := status["refname"].(string)
		if !configuration.MatchesAnyPredicate(config.Refs, refname) {
			return http.StatusOK, nil, nil
		}
	}

	if err := ctx.Events.Emit("bitbucket.commitStatus", data); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil, nil
}

func (s *OnCommitStatus) Cleanup(ctx core.TriggerContext) error {
	return nil
}
