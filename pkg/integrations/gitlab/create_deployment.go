package gitlab

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

//go:embed example_output_create_deployment.json
var exampleOutputCreateDeployment []byte

const (
	DeploymentPayloadType = "gitlab.deployment"

	DeploymentStatusRunning  = "running"
	DeploymentStatusSuccess  = "success"
	DeploymentStatusFailed   = "failed"
	DeploymentStatusCanceled = "canceled"
)

// deploymentStatuses lists the statuses accepted by the GitLab deployments API.
var deploymentStatuses = []string{
	DeploymentStatusRunning,
	DeploymentStatusSuccess,
	DeploymentStatusFailed,
	DeploymentStatusCanceled,
}

// deploymentStatusOptions builds the select options shared by the deployment components.
func deploymentStatusOptions() []configuration.FieldOption {
	return []configuration.FieldOption{
		{Label: "Running", Value: DeploymentStatusRunning},
		{Label: "Success", Value: DeploymentStatusSuccess},
		{Label: "Failed", Value: DeploymentStatusFailed},
		{Label: "Canceled", Value: DeploymentStatusCanceled},
	}
}

type CreateDeployment struct{}

type CreateDeploymentConfiguration struct {
	Project     string `mapstructure:"project"`
	Environment string `mapstructure:"environment"`
	Ref         string `mapstructure:"ref"`
	SHA         string `mapstructure:"sha"`
	Tag         bool   `mapstructure:"tag"`
	Status      string `mapstructure:"status"`
}

func (c *CreateDeployment) Name() string {
	return "gitlab.createDeployment"
}

func (c *CreateDeployment) Label() string {
	return "Create Deployment"
}

func (c *CreateDeployment) Description() string {
	return "Create a deployment for a GitLab environment"
}

func (c *CreateDeployment) Documentation() string {
	return `The Create Deployment component records a deployment for a GitLab project environment.

## Use Cases

- **Deployment tracking**: Record deployments performed outside of GitLab CI/CD so they show up in the environment's deployment history
- **Release automation**: Mark a commit as deployed to an environment (staging, production) from a SuperPlane workflow
- **Status coordination**: Pair with **Create Deployment Status** to transition the deployment as your rollout progresses

## Configuration

- **Project** (required): The GitLab project to deploy in
- **Environment** (required): The target environment (e.g., production, staging). Pick an existing one from the dropdown, or switch to Expression and type a new name - GitLab creates it automatically on first deploy.
- **Ref** (required): The branch or tag being deployed (defaults to main). When you pick **Tag** in this field, the deployment is automatically recorded as a tag deployment.
- **Commit SHA** (required): The commit SHA being deployed. Supports expressions.
- **Ref is a tag**: Only needed when **Ref** comes from an expression that doesn't carry the tag prefix (e.g. a raw tag name). Not needed when using the Ref field's own Tag option.
- **Status**: The initial deployment status (defaults to running)

## Output

Returns the created deployment object, including:
- **id**: The deployment ID (use it with Create Deployment Status)
- **iid**: The project-relative deployment ID
- **status**: The deployment status
- **environment**: The environment the deployment targets

## Requirements

The connected user needs at least the **Developer** role on the project, and for protected environments must be in the environment's **Allowed to deploy** list.`
}

func (c *CreateDeployment) Icon() string {
	return "gitlab"
}

func (c *CreateDeployment) Color() string {
	return "orange"
}

func (c *CreateDeployment) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateDeployment) ExampleOutput() map[string]any {
	var example map[string]any
	if err := json.Unmarshal(exampleOutputCreateDeployment, &example); err != nil {
		return map[string]any{}
	}
	return example
}

func (c *CreateDeployment) Configuration() []configuration.Field {
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
			Name:        "environment",
			Label:       "Environment",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The target environment. Pick an existing one, or if it doesn't exist yet, switch to Expression and type its name - GitLab creates it automatically on first deploy.",
			Placeholder: "e.g. production",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           ResourceTypeEnvironment,
					UseNameAsValue: true,
					Parameters: []configuration.ParameterRef{
						{
							Name:      "project",
							ValueFrom: &configuration.ParameterValueFrom{Field: "project"},
						},
					},
				},
			},
		},
		{
			Name:     "ref",
			Label:    "Ref",
			Type:     configuration.FieldTypeGitRef,
			Required: true,
			Default:  "main",
		},
		{
			Name:        "sha",
			Label:       "Commit SHA",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The commit SHA being deployed. Supports expressions.",
		},
		{
			Name:        "tag",
			Label:       "Ref is a tag",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     false,
			Description: "Only needed when Ref comes from an expression without the tag prefix. Automatically detected when you pick Tag in the Ref field.",
		},
		{
			Name:     "status",
			Label:    "Status",
			Type:     configuration.FieldTypeSelect,
			Required: false,
			Default:  DeploymentStatusRunning,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: deploymentStatusOptions(),
				},
			},
		},
	}
}

func (c *CreateDeployment) Setup(ctx core.SetupContext) error {
	var config CreateDeploymentConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Project == "" {
		return fmt.Errorf("project is required")
	}

	if strings.TrimSpace(config.Environment) == "" {
		return fmt.Errorf("environment is required")
	}

	if strings.TrimSpace(config.Ref) == "" {
		return fmt.Errorf("ref is required")
	}

	if strings.TrimSpace(config.SHA) == "" {
		return fmt.Errorf("commit SHA is required")
	}

	if config.Status != "" && !slices.Contains(deploymentStatuses, config.Status) {
		return fmt.Errorf("invalid status %q: must be one of running, success, failed, canceled", config.Status)
	}

	return ensureProjectInMetadata(
		ctx.Metadata,
		ctx.Integration,
		config.Project,
	)
}

func (c *CreateDeployment) Execute(ctx core.ExecutionContext) error {
	var config CreateDeploymentConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to initialize GitLab client: %w", err)
	}

	status := config.Status
	if status == "" {
		status = DeploymentStatusRunning
	}

	deployment, err := client.CreateDeployment(context.Background(), config.Project, &CreateDeploymentRequest{
		Environment: config.Environment,
		Ref:         normalizePipelineRef(config.Ref),
		SHA:         config.SHA,
		Tag:         config.Tag || isTagRef(config.Ref),
		Status:      status,
	})
	if err != nil {
		return fmt.Errorf("failed to create deployment: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		DeploymentPayloadType,
		[]any{deployment},
	)
}

func (c *CreateDeployment) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateDeployment) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}

func (c *CreateDeployment) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateDeployment) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *CreateDeployment) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *CreateDeployment) HandleHook(ctx core.ActionHookContext) error {
	return nil
}

// isTagRef reports whether ref carries the refs/tags/ prefix produced by the
// git-ref field's "Tag" option, so Tag is inferred automatically instead of
// relying solely on the user also toggling "Ref is a tag" by hand.
func isTagRef(ref string) bool {
	return strings.HasPrefix(ref, "refs/tags/")
}
