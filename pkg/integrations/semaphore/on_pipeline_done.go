package semaphore

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
)

type OnPipelineDone struct{}

type OnPipelineDoneMetadata struct {
	Project *Project `json:"project"`
}

var AllPipelineDoneResults = []configuration.FieldOption{
	{Label: "Passed", Value: "passed"},
	{Label: "Failed", Value: "failed"},
	{Label: "Stopped", Value: "stopped"},
	{Label: "Canceled", Value: "canceled"},
}

type OnPipelineDoneConfiguration struct {
	Project   string                    `json:"project" mapstructure:"project"`
	Refs      []configuration.Predicate `json:"refs" mapstructure:"refs"`
	Results   []string                  `json:"results" mapstructure:"results"`
	Pipelines []configuration.Predicate `json:"pipelines" mapstructure:"pipelines"`
}

func (p *OnPipelineDone) Name() string {
	return "semaphore.onPipelineDone"
}

func (p *OnPipelineDone) Label() string {
	return "On Pipeline Done"
}

func (p *OnPipelineDone) Description() string {
	return "Listen to Semaphore pipeline done events"
}

func (p *OnPipelineDone) Documentation() string {
	return `The On Pipeline Done trigger starts a workflow execution when a Semaphore pipeline completes.

## Use Cases

- **Pipeline orchestration**: Chain workflows together based on pipeline completion
- **Status monitoring**: Monitor CI/CD pipeline results
- **Notification workflows**: Send notifications when pipelines succeed or fail
- **Post-processing**: Process artifacts or results after pipeline completion

## Configuration

- **Project**: Select the Semaphore project to monitor
- **Refs**: Optional ref filters (for example ` + "`refs/heads/main`" + `)
- **Results**: Optional pipeline result filters (for example ` + "`passed`" + `, ` + "`failed`" + `)
- **Pipelines**: Optional pipeline file filters (for example ` + "`.semaphore/semaphore.yml`" + `, ` + "`.semaphore/production/deploy.yml`" + `)

## Event Data

Each pipeline done event includes:
- **pipeline**: Pipeline information including ID, state, and result
- **workflow**: Workflow information including ID and URL
- **project**: Project information
- **result**: Pipeline result (passed, failed, stopped, etc.)
- **state**: Pipeline state (done)

## Webhook Setup

This trigger automatically sets up a Semaphore webhook when configured. The webhook is managed by SuperPlane and will be cleaned up when the trigger is removed.`
}

func (p *OnPipelineDone) Icon() string {
	return "workflow"
}

func (p *OnPipelineDone) Color() string {
	return "gray"
}

func (p *OnPipelineDone) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "project",
			Label:    "Project",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "project",
					UseNameAsValue: true,
				},
			},
		},
		{
			Name:     "results",
			Label:    "Results",
			Type:     configuration.FieldTypeMultiSelect,
			Required: false,
			Default:  []string{"passed"},
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: AllPipelineDoneResults,
				},
			},
		},
		{
			Name:     "refs",
			Label:    "Refs",
			Type:     configuration.FieldTypeAnyPredicateList,
			Required: false,
			Default:  []map[string]any{{"type": configuration.PredicateTypeEquals, "value": "refs/heads/main"}},
			TypeOptions: &configuration.TypeOptions{
				AnyPredicateList: &configuration.AnyPredicateListTypeOptions{
					Operators: configuration.AllPredicateOperators,
				},
			},
		},
		{
			Name:     "pipelines",
			Label:    "Pipelines",
			Type:     configuration.FieldTypeAnyPredicateList,
			Required: false,
			Default:  []map[string]any{{"type": configuration.PredicateTypeEquals, "value": ".semaphore/semaphore.yml"}},
			TypeOptions: &configuration.TypeOptions{
				AnyPredicateList: &configuration.AnyPredicateListTypeOptions{
					Operators: configuration.AllPredicateOperators,
				},
			},
		},
	}
}

func (p *OnPipelineDone) Setup(ctx core.TriggerContext) error {
	var metadata OnPipelineDoneMetadata
	err := mapstructure.Decode(ctx.Metadata.Get(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to parse metadata: %w", err)
	}

	//
	// If metadata is set, it means the trigger was already setup
	//
	if metadata.Project != nil {
		return nil
	}

	config := OnPipelineDoneConfiguration{}
	err = mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Project == "" {
		return fmt.Errorf("project is required")
	}

	//
	// If this is the same project, nothing to do.
	//
	if metadata.Project != nil && (config.Project == metadata.Project.ID || config.Project == metadata.Project.Name) {
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	project, err := client.GetProject(config.Project)
	if err != nil {
		return fmt.Errorf("error finding project %s: %v", config.Project, err)
	}

	err = ctx.Metadata.Set(OnPipelineDoneMetadata{
		Project: &Project{
			ID:   project.Metadata.ProjectID,
			Name: project.Metadata.ProjectName,
			URL:  fmt.Sprintf("%s/projects/%s", string(client.OrgURL), project.Metadata.ProjectID),
		},
	})

	if err != nil {
		return fmt.Errorf("error setting metadata: %v", err)
	}

	return ctx.Integration.RequestWebhook(WebhookConfiguration{
		Project: project.Metadata.ProjectName,
	})
}

func (p *OnPipelineDone) Actions() []core.Action {
	return []core.Action{}
}

func (p *OnPipelineDone) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (p *OnPipelineDone) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	config := OnPipelineDoneConfiguration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to decode configuration: %w", err)
	}

	signature := ctx.Headers.Get("X-Semaphore-Signature-256")
	if signature == "" {
		return http.StatusForbidden, fmt.Errorf("invalid signature")
	}

	signature = strings.TrimPrefix(signature, "sha256=")
	if signature == "" {
		return http.StatusForbidden, fmt.Errorf("invalid signature")
	}

	secret, err := ctx.Webhook.GetSecret()
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error authenticating request")
	}

	if err := crypto.VerifySignature(secret, ctx.Body, signature); err != nil {
		return http.StatusForbidden, fmt.Errorf("invalid signature")
	}

	payload := map[string]any{}
	err = json.Unmarshal(ctx.Body, &payload)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("error parsing request body: %v", err)
	}

	if len(config.Refs) > 0 {
		ref, ok := getNestedString(payload, "revision", "reference")
		if !ok || strings.TrimSpace(ref) == "" {
			return http.StatusBadRequest, fmt.Errorf("missing revision.reference")
		}

		if !configuration.MatchesAnyPredicate(config.Refs, ref) {
			ctx.Logger.Infof("ref %s does not match the allowed predicates: %v", ref, config.Refs)
			return http.StatusOK, nil
		}
	}

	if len(config.Results) > 0 {
		result, ok := getNestedString(payload, "pipeline", "result")
		if !ok || strings.TrimSpace(result) == "" {
			return http.StatusBadRequest, fmt.Errorf("missing pipeline.result")
		}

		if !matchesPipelineResult(config.Results, result) {
			ctx.Logger.Infof("result %s does not match the allowed predicates: %v", result, config.Results)
			return http.StatusOK, nil
		}
	}

	if len(config.Pipelines) > 0 {
		workingDirectory, ok := getNestedString(payload, "pipeline", "working_directory")
		if !ok || strings.TrimSpace(workingDirectory) == "" {
			return http.StatusBadRequest, fmt.Errorf("missing pipeline.working_directory")
		}

		pipelineFile, ok := getNestedString(payload, "pipeline", "yaml_file_name")
		if !ok || strings.TrimSpace(pipelineFile) == "" {
			return http.StatusBadRequest, fmt.Errorf("missing pipeline.yaml_file_name")
		}

		pipelinePath := fmt.Sprintf("%s/%s", workingDirectory, pipelineFile)
		if !configuration.MatchesAnyPredicate(config.Pipelines, pipelinePath) {
			ctx.Logger.Infof("pipeline file %s does not match the allowed predicates: %v", pipelinePath, config.Pipelines)
			return http.StatusOK, nil
		}
	}

	err = ctx.Events.Emit("semaphore.pipeline.done", payload)

	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil
}

func (p *OnPipelineDone) Cleanup(ctx core.TriggerContext) error {
	return nil
}

func getNestedString(payload map[string]any, keys ...string) (string, bool) {
	current := any(payload)

	for _, key := range keys {
		obj, ok := current.(map[string]any)
		if !ok {
			return "", false
		}

		next, ok := obj[key]
		if !ok {
			return "", false
		}

		current = next
	}

	result, ok := current.(string)
	return result, ok
}

func matchesPipelineResult(allowedResults []string, result string) bool {
	normalizedResult := normalizePipelineResult(result)

	return slices.ContainsFunc(allowedResults, func(allowed string) bool {
		return normalizePipelineResult(allowed) == normalizedResult
	})
}

func normalizePipelineResult(result string) string {
	normalized := strings.ToLower(strings.TrimSpace(result))
	if normalized == "cancelled" {
		return "canceled"
	}

	return normalized
}
