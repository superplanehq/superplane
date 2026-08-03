package gitlab

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnCommitStatus struct{}

type OnCommitStatusConfiguration struct {
	Project  string                    `json:"project" mapstructure:"project"`
	Statuses []string                  `json:"statuses" mapstructure:"statuses"`
	Names    []configuration.Predicate `json:"names" mapstructure:"names"`
	Refs     []configuration.Predicate `json:"refs" mapstructure:"refs"`
}

func (t *OnCommitStatus) Name() string {
	return "gitlab.onCommitStatus"
}

func (t *OnCommitStatus) Label() string {
	return "On Commit Status"
}

func (t *OnCommitStatus) Description() string {
	return "Listen to commit status changes in a GitLab project"
}

func (t *OnCommitStatus) Documentation() string {
	return `The On Commit Status trigger starts a workflow execution when the CI status of a commit changes in a GitLab project.

GitLab has no dedicated commit status webhook: a commit's statuses are jobs grouped under a pipeline, and statuses published through the commit status API land in a pipeline of their own (` + "`source: external`" + `). This trigger is therefore backed by GitLab's pipeline events, and each event carries the commit, its rolled-up status, and every individual status in ` + "`builds`" + `.

## Use Cases

- **Merge gates**: React when a commit turns green or red before merging
- **External CI fan-in**: Respond to statuses published by external systems, including SuperPlane's own Publish Commit Status component
- **Targeted checks**: Watch a single named status, such as ` + "`security-scan`" + `, instead of the whole pipeline
- **Notifications**: Alert a team when a protected branch's commit status fails

## Configuration

- **Project** (required): GitLab project to monitor
- **Statuses** (required): Commit status states to listen for. Matched against the commit's rolled-up status. Default: success, failed.
- **Names** *(optional)*: Filter on the names of the statuses carried by the event, e.g. equals ` + "`security-scan`" + ` or matches ` + "`deploy/.*`" + `. The event is forwarded when any status name matches.
- **Refs** *(optional)*: Filter on the branch or tag the statuses were reported for.

## Event Data

Each event includes:
- **object_attributes**: The commit's rolled-up status, plus ref, sha, source, stages, duration, and a link to the pipeline
- **builds**: Every individual commit status, each with its name, stage, status, and timings
- **commit**: The commit the statuses belong to
- **project**: Project information
- **user**: The user the statuses are attributed to
- **merge_request**: The merge request the commit belongs to, when there is one

## Permissions

The connected token needs the ` + "`api`" + ` scope, and the connected user needs at least the **Maintainer** role on the project so SuperPlane can create the webhook.

## Webhook Setup

This trigger automatically sets up a GitLab webhook when configured. The webhook is managed by SuperPlane and will be cleaned up when the trigger is removed.`
}

func (t *OnCommitStatus) Icon() string {
	return "gitlab"
}

func (t *OnCommitStatus) Color() string {
	return "orange"
}

func (t *OnCommitStatus) Configuration() []configuration.Field {
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
			Name:     "statuses",
			Label:    "Statuses",
			Type:     configuration.FieldTypeMultiSelect,
			Required: true,
			Default:  []string{CommitStatusStateSuccess, CommitStatusStateFailed},
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: commitStatusStateOptions(),
				},
			},
		},
		{
			Name:        "names",
			Label:       "Names",
			Description: "Optional. Filter the commit status names carried by the event, e.g. security-scan or deploy/.*",
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
			Name:        "refs",
			Label:       "Refs",
			Description: "Optional. Filter the branch or tag the statuses were reported for.",
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
	var config OnCommitStatusConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if err := ensureProjectInMetadata(ctx.Metadata, ctx.Integration, config.Project); err != nil {
		return err
	}

	return ctx.Integration.RequestWebhook(WebhookConfiguration{
		EventType: "pipeline",
		ProjectID: config.Project,
	})
}

func (t *OnCommitStatus) Hooks() []core.Hook {
	return []core.Hook{}
}

func (t *OnCommitStatus) HandleHook(ctx core.TriggerHookContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnCommitStatus) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	var config OnCommitStatusConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("failed to decode configuration: %w", err)
	}

	eventType := ctx.Headers.Get("X-Gitlab-Event")
	if eventType == "" {
		return http.StatusBadRequest, nil, fmt.Errorf("missing X-Gitlab-Event header")
	}

	if eventType != "Pipeline Hook" {
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

	attributes, ok := data["object_attributes"].(map[string]any)
	if !ok {
		return http.StatusBadRequest, nil, fmt.Errorf("object_attributes missing from pipeline payload")
	}

	status, ok := attributes["status"].(string)
	if !ok {
		return http.StatusBadRequest, nil, fmt.Errorf("status missing from pipeline payload")
	}

	if len(config.Statuses) > 0 && !slices.Contains(config.Statuses, status) {
		ctx.Logger.Infof("Ignoring event - status %s is not in the allowed list: %v", status, config.Statuses)
		return http.StatusOK, nil, nil
	}

	ref, _ := attributes["ref"].(string)
	if len(config.Refs) > 0 && !configuration.MatchesAnyPredicate(config.Refs, ref) {
		ctx.Logger.Infof("Ignoring event - ref %s does not match the allowed predicates: %v", ref, config.Refs)
		return http.StatusOK, nil, nil
	}

	if len(config.Names) > 0 && !configuration.MatchesAnyPredicateInList(config.Names, commitStatusNames(data)) {
		ctx.Logger.Infof("Ignoring event - no commit status name matches the allowed predicates: %v", config.Names)
		return http.StatusOK, nil, nil
	}

	if err := ctx.Events.Emit("gitlab.commitStatusChanged", data); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil, nil
}

func (t *OnCommitStatus) Cleanup(ctx core.TriggerContext) error {
	return nil
}

// commitStatusNames collects the name of every commit status carried by a
// pipeline payload. GitLab reports each status as an entry in "builds",
// including the ones published through the commit status API.
func commitStatusNames(data map[string]any) []string {
	builds, ok := data["builds"].([]any)
	if !ok {
		return nil
	}

	names := make([]string, 0, len(builds))
	for _, item := range builds {
		build, ok := item.(map[string]any)
		if !ok {
			continue
		}

		if name, ok := build["name"].(string); ok {
			names = append(names, name)
		}
	}

	return names
}
