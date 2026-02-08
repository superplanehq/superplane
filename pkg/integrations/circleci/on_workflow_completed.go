package circleci

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnWorkflowCompleted struct{}

type OnWorkflowCompletedConfiguration struct {
	ProjectSlug string `json:"projectSlug" mapstructure:"projectSlug"`
}

type OnWorkflowCompletedMetadata struct {
	Project *Project `json:"project" mapstructure:"project"`
}

func (t *OnWorkflowCompleted) Name() string        { return "circleci.onWorkflowCompleted" }
func (t *OnWorkflowCompleted) Label() string       { return "On Workflow Completed" }
func (t *OnWorkflowCompleted) Description() string { return "Start a workflow when a CircleCI workflow completes" }

func (t *OnWorkflowCompleted) Documentation() string {
	return `The On Workflow Completed trigger starts a workflow execution when a CircleCI workflow completes.

This trigger provisions an outbound webhook in CircleCI (scope: project) and validates incoming requests using the
` + "`" + `circleci-signature` + "`" + ` header (HMAC-SHA256 over request body; v1).
`
}

func (t *OnWorkflowCompleted) Icon() string  { return "workflow" }
func (t *OnWorkflowCompleted) Color() string { return "gray" }

func (t *OnWorkflowCompleted) ExampleData() map[string]any {
	return map[string]any{
		"type": "workflow-completed",
		"workflow": map[string]any{
			"id": "5034460f-c7c4-4c43-9457-de07e2029e7b",
		},
	}
}

func (t *OnWorkflowCompleted) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "projectSlug",
			Label:       "Project Slug",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "e.g. gh/my-org/my-repo",
			Description: "CircleCI project slug in the form vcs-slug/org-name/repo-name",
		},
	}
}

func (t *OnWorkflowCompleted) Setup(ctx core.TriggerContext) error {
	metadata := OnWorkflowCompletedMetadata{}
	_ = mapstructure.Decode(ctx.Metadata.Get(), &metadata)
	if metadata.Project != nil {
		return nil
	}

	cfg := OnWorkflowCompletedConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &cfg); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}
	if cfg.ProjectSlug == "" {
		return errors.New("projectSlug is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}
	project, err := client.GetProjectBySlug(cfg.ProjectSlug)
	if err != nil {
		return fmt.Errorf("failed to fetch project: %w", err)
	}

	if err := ctx.Metadata.Set(OnWorkflowCompletedMetadata{Project: project}); err != nil {
		return fmt.Errorf("failed to set metadata: %w", err)
	}

	return ctx.Integration.RequestWebhook(WebhookConfiguration{
		ProjectID:   project.ID,
		ProjectSlug: project.Slug,
		Events:      []string{"workflow-completed"},
	})
}

func (t *OnWorkflowCompleted) Actions() []core.Action { return []core.Action{} }
func (t *OnWorkflowCompleted) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func extractV1Signature(header string) string {
	// Header is a comma-separated list like: "v1=<hex>,v2=...,v3=..."
	parts := strings.Split(header, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "v1=") {
			return strings.TrimPrefix(part, "v1=")
		}
	}
	return ""
}

func (t *OnWorkflowCompleted) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	sigHeader := ctx.Headers.Get("circleci-signature")
	if sigHeader == "" {
		sigHeader = ctx.Headers.Get("Circleci-Signature")
	}
	if sigHeader == "" {
		return http.StatusForbidden, fmt.Errorf("missing circleci-signature header")
	}

	sig := extractV1Signature(sigHeader)
	if sig == "" {
		return http.StatusForbidden, fmt.Errorf("missing v1 signature")
	}

	secret, err := ctx.Webhook.GetSecret()
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to get webhook secret")
	}

	mac := hmac.New(sha256.New, secret)
	mac.Write(ctx.Body)
	expected := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(sig)) {
		return http.StatusForbidden, fmt.Errorf("invalid signature")
	}

	payload := map[string]any{}
	if err := json.Unmarshal(ctx.Body, &payload); err != nil {
		return http.StatusBadRequest, fmt.Errorf("failed to parse JSON body: %w", err)
	}

	// Emit a single payload type for now.
	if err := ctx.Events.Emit("circleci.workflow.completed", payload); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to emit event: %w", err)
	}

	return http.StatusOK, nil
}

func (t *OnWorkflowCompleted) Cleanup(ctx core.TriggerContext) error { return nil }

