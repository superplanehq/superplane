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

const (
	JobStatusCreated  = "created"
	JobStatusPending  = "pending"
	JobStatusRunning  = "running"
	JobStatusSuccess  = "success"
	JobStatusFailed   = "failed"
	JobStatusCanceled = "canceled"
	JobStatusSkipped  = "skipped"
	JobStatusManual   = "manual"
)

func jobStatusOptions() []configuration.FieldOption {
	return []configuration.FieldOption{
		{Label: "Created", Value: JobStatusCreated},
		{Label: "Pending", Value: JobStatusPending},
		{Label: "Running", Value: JobStatusRunning},
		{Label: "Success", Value: JobStatusSuccess},
		{Label: "Failed", Value: JobStatusFailed},
		{Label: "Canceled", Value: JobStatusCanceled},
		{Label: "Skipped", Value: JobStatusSkipped},
		{Label: "Manual", Value: JobStatusManual},
	}
}

type OnJob struct{}

type OnJobConfiguration struct {
	Project  string                    `json:"project" mapstructure:"project"`
	Statuses []string                  `json:"statuses" mapstructure:"statuses"`
	Names    []configuration.Predicate `json:"names" mapstructure:"names"`
	Refs     []configuration.Predicate `json:"refs" mapstructure:"refs"`
}

func (t *OnJob) Name() string {
	return "gitlab.onJob"
}

func (t *OnJob) Label() string {
	return "On Job"
}

func (t *OnJob) Description() string {
	return "Listen to job events from GitLab"
}

func (t *OnJob) Documentation() string {
	return `The On Job trigger starts a workflow execution when the status of a CI/CD job changes in a GitLab project.

Jobs are the individual units of work inside a pipeline. Where the On Pipeline trigger fires once for the pipeline as a whole, this trigger fires for each job, so you can react to one specific job - a deploy, a scan, a migration - without waiting for the rest of the pipeline.

## Use Cases

- **Per-job automation**: Run a workflow the moment the ` + "`deploy`" + ` job succeeds, without waiting for the whole pipeline
- **Targeted failure handling**: Page or open an issue when a named job fails
- **Stage gates**: React when a job in a specific stage finishes
- **Build observability**: Record durations, runners, and failure reasons per job

## Configuration

- **Project** (required): GitLab project to monitor
- **Statuses** (required): Job statuses to listen for. Default: success, failed.
- **Names** *(optional)*: Filter on the job name, e.g. equals ` + "`deploy`" + ` or matches ` + "`test-.*`" + `
- **Refs** *(optional)*: Filter on the branch or tag the job ran for

## Event Data

Each event includes:
- **build_name**, **build_stage**, **build_status**: The job, its stage, and its new status
- **build_started_at**, **build_finished_at**, **build_duration**, **build_queued_duration**: Job timings
- **build_failure_reason**, **build_allow_failure**, **retries_count**: Failure and retry details
- **pipeline_id**, **ref**, **sha**, **tag**: The pipeline and ref the job belongs to
- **runner**: The runner that executed the job
- **commit**, **project**, **repository**, **user**, **environment**: Surrounding context

Note that GitLab excludes trigger jobs from this event, and commit statuses published by external systems are not jobs, so they do not produce job events.

## Permissions

The connected token needs the ` + "`api`" + ` scope, and the connected user needs at least the **Maintainer** role on the project so SuperPlane can create the webhook.

## Webhook Setup

This trigger automatically sets up a GitLab webhook when configured. The webhook is managed by SuperPlane and will be cleaned up when the trigger is removed.`
}

func (t *OnJob) Icon() string {
	return "gitlab"
}

func (t *OnJob) Color() string {
	return "orange"
}

func (t *OnJob) Configuration() []configuration.Field {
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
			Default:  []string{JobStatusSuccess, JobStatusFailed},
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: jobStatusOptions(),
				},
			},
		},
		{
			Name:        "names",
			Label:       "Names",
			Description: "Optional. Filter the job name, e.g. deploy or test-.*",
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
			Description: "Optional. Filter the branch or tag the job ran for.",
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

func (t *OnJob) Setup(ctx core.TriggerContext) error {
	var config OnJobConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if err := ensureProjectInMetadata(ctx.Metadata, ctx.Integration, config.Project); err != nil {
		return err
	}

	return ctx.Integration.RequestWebhook(WebhookConfiguration{
		EventType: "job",
		ProjectID: config.Project,
	})
}

func (t *OnJob) Hooks() []core.Hook {
	return []core.Hook{}
}

func (t *OnJob) HandleHook(ctx core.TriggerHookContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnJob) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	var config OnJobConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("failed to decode configuration: %w", err)
	}

	eventType := ctx.Headers.Get("X-Gitlab-Event")
	if eventType == "" {
		return http.StatusBadRequest, nil, fmt.Errorf("missing X-Gitlab-Event header")
	}

	if eventType != "Job Hook" {
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

	status, ok := data["build_status"].(string)
	if !ok {
		return http.StatusBadRequest, nil, fmt.Errorf("build_status missing from job payload")
	}

	if len(config.Statuses) > 0 && !slices.Contains(config.Statuses, status) {
		ctx.Logger.Infof("Ignoring event - status %s is not in the allowed list: %v", status, config.Statuses)
		return http.StatusOK, nil, nil
	}

	name, _ := data["build_name"].(string)
	if len(config.Names) > 0 && !configuration.MatchesAnyPredicate(config.Names, name) {
		ctx.Logger.Infof("Ignoring event - job %s does not match the allowed predicates: %v", name, config.Names)
		return http.StatusOK, nil, nil
	}

	ref, _ := data["ref"].(string)
	if len(config.Refs) > 0 && !configuration.MatchesAnyPredicate(config.Refs, ref) {
		ctx.Logger.Infof("Ignoring event - ref %s does not match the allowed predicates: %v", ref, config.Refs)
		return http.StatusOK, nil, nil
	}

	if err := ctx.Events.Emit("gitlab.job", data); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil, nil
}

func (t *OnJob) Cleanup(ctx core.TriggerContext) error {
	return nil
}
