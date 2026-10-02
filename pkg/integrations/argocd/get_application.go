package argocd

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const GetApplicationPayloadType = "argocd.application"

type GetApplication struct{}

type GetApplicationConfiguration struct {
	Project              string `json:"project" mapstructure:"project"`
	Application          string `json:"application" mapstructure:"application"`
	ApplicationNamespace string `json:"applicationNamespace" mapstructure:"applicationNamespace"`
}

func (c *GetApplication) Name() string {
	return "argocd.getApplication"
}

func (c *GetApplication) Label() string {
	return "Get Application"
}

func (c *GetApplication) Description() string {
	return "Get the current delivery state for an Argo CD application"
}

func (c *GetApplication) Documentation() string {
	return `The Get Application component reads the current state of one Argo CD application.

## Configuration

- **Project**: Argo CD project that contains the application
- **Application**: Argo CD application name
- **Application Namespace**: Optional application namespace for installations that allow applications in any namespace

## Output

Emits an ` + "`argocd.application`" + ` payload with application identity, sources, destination, sync state, health state, operation state, and conditions.`
}

func (c *GetApplication) Icon() string {
	return "kubernetes"
}

func (c *GetApplication) Color() string {
	return "gray"
}

func (c *GetApplication) ExampleOutput() map[string]any {
	return map[string]any{
		"application": map[string]any{
			"name":      "payments",
			"namespace": "argocd",
			"uid":       "app-1",
			"project":   "platform",
		},
		"sources": []map[string]any{{
			"repoURL":        "https://github.com/example/platform.git",
			"path":           "apps/payments",
			"targetRevision": "main",
		}},
		"destination": map[string]any{
			"server":    "https://kubernetes.default.svc",
			"namespace": "payments",
		},
		"sync": map[string]any{
			"status":   "Synced",
			"revision": "abc123",
		},
		"health": map[string]any{
			"status": "Healthy",
		},
	}
}

func (c *GetApplication) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetApplication) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "project",
			Label:       "Project",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Argo CD project that contains the application",
		},
		{
			Name:        "application",
			Label:       "Application",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Argo CD application name",
		},
		{
			Name:        "applicationNamespace",
			Label:       "Application Namespace",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Togglable:   true,
			Description: "Application namespace when Argo CD allows applications in any namespace",
		},
	}
}

func (c *GetApplication) Setup(ctx core.SetupContext) error {
	_, err := decodeGetApplicationConfiguration(ctx.Configuration)
	return err
}

func (c *GetApplication) Execute(ctx core.ExecutionContext) error {
	configuration, err := decodeGetApplicationConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	application, err := client.GetApplication(configuration.Project, configuration.Application, configuration.ApplicationNamespace)
	if err != nil {
		return err
	}

	if strings.TrimSpace(application.Metadata.Name) == "" {
		return fmt.Errorf("Argo CD response missing application metadata.name")
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		GetApplicationPayloadType,
		[]any{applicationOutputFrom(application, configuration.Project)},
	)
}

func (c *GetApplication) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *GetApplication) HandleHook(ctx core.ActionHookContext) error {
	return nil
}

func (c *GetApplication) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *GetApplication) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetApplication) Cleanup(ctx core.SetupContext) error {
	return nil
}

func decodeGetApplicationConfiguration(configuration any) (GetApplicationConfiguration, error) {
	spec := GetApplicationConfiguration{}
	if err := mapstructure.Decode(configuration, &spec); err != nil {
		return GetApplicationConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}

	spec.Project = strings.TrimSpace(spec.Project)
	if spec.Project == "" {
		return GetApplicationConfiguration{}, fmt.Errorf("project is required")
	}

	spec.Application = strings.TrimSpace(spec.Application)
	if spec.Application == "" {
		return GetApplicationConfiguration{}, fmt.Errorf("application is required")
	}

	spec.ApplicationNamespace = strings.TrimSpace(spec.ApplicationNamespace)
	return spec, nil
}
