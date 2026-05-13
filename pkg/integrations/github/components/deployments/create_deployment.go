package deployments

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/go-github/v84/github"
	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
)

type CreateDeployment struct{}

type CreateDeploymentConfiguration struct {
	Repository            string   `json:"repository" mapstructure:"repository"`
	Ref                   string   `json:"ref" mapstructure:"ref"`
	Environment           string   `json:"environment" mapstructure:"environment"`
	Description           string   `json:"description" mapstructure:"description"`
	Task                  string   `json:"task" mapstructure:"task"`
	TransientEnvironment  bool     `json:"transientEnvironment" mapstructure:"transientEnvironment"`
	ProductionEnvironment bool     `json:"productionEnvironment" mapstructure:"productionEnvironment"`
	AutoMerge             bool     `json:"autoMerge" mapstructure:"autoMerge"`
	RequiredContexts      []string `json:"requiredContexts" mapstructure:"requiredContexts"`
}

func (c *CreateDeployment) Name() string {
	return "github.createDeployment"
}

func (c *CreateDeployment) Label() string {
	return "Create Deployment"
}

func (c *CreateDeployment) Description() string {
	return "Create a GitHub deployment for a ref and environment (enables the PR deployment UI)"
}

func (c *CreateDeployment) Documentation() string {
	return `The Create Deployment component registers a deployment with GitHub for a given ref and environment. This drives the **View deployment** button and environment box on pull requests—the same surface area used by Railway, Vercel, and Netlify.

## Use Cases

- **Preview environments**: Create a deployment when a PR opens, then post statuses after provision/build
- **Environment history**: Let GitHub track deployment activity per environment name
- **Ephemeral previews**: Set **Transient environment** so GitHub treats the deployment as short-lived

## Configuration

- **Repository**: GitHub repository for the deployment
- **Ref**: Branch name, tag, or commit SHA to deploy
- **Environment**: Environment name (for example preview-pr-42)
- **Description**: Optional deployment description
- **Task**: Optional deployment task (GitHub default is deploy if omitted)
- **Transient environment**: Mark as ephemeral so GitHub can auto-clean inactive deployments
- **Production environment**: Mark as production when applicable
- **Auto merge** (default off): Whether GitHub may auto-merge the default branch into the ref if needed
- **Required contexts**: Status check context names that must pass before the deployment is created (same names as commit status **context**). Leave empty to skip status gating (recommended for previews, avoids HTTP 409 when CI is pending). If you list one or more contexts, only those must be green. GitHub may respond with HTTP 202 while it prepares verification; this component retries automatically for a short period

## Output

Emits github.deployment with the created deployment, including id for **Create Deployment Status**.`
}

func (c *CreateDeployment) Icon() string {
	return "github"
}

func (c *CreateDeployment) Color() string {
	return "gray"
}

func (c *CreateDeployment) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateDeployment) Configuration() []configuration.Field {
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
			Name:        "ref",
			Label:       "Ref",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "e.g. feature/my-branch or {{event.data.pull_request.head.ref}}",
			Description: "Branch name, tag, or full commit SHA to associate with the deployment",
		},
		{
			Name:        "environment",
			Label:       "Environment",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "e.g. preview-pr-{{event.data.pull_request.number}}",
			Description: "Deployment environment name shown in GitHub",
		},
		{
			Name:        "description",
			Label:       "Description",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Placeholder: "Optional short description",
		},
		{
			Name:        "task",
			Label:       "Task",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Placeholder: "deploy (default if empty)",
			Description: "Deployment task identifier",
		},
		{
			Name:        "transientEnvironment",
			Label:       "Transient environment",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     false,
			Description: "Ephemeral environment; GitHub may clean up inactive deployments",
		},
		{
			Name:        "productionEnvironment",
			Label:       "Production environment",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     false,
			Description: "Mark this deployment as targeting production",
		},
		{
			Name:        "autoMerge",
			Label:       "Auto merge",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     false,
			Description: "Allow GitHub to merge the default branch into the ref if needed",
		},
		{
			Name:        "requiredContexts",
			Label:       "Required contexts",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Description: "Commit status contexts that must pass before creating the deployment; leave empty to skip (preview-friendly)",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Context",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeString,
					},
				},
			},
		},
	}
}

func (c *CreateDeployment) Setup(ctx core.SetupContext) error {
	return common.EnsureRepoInMetadata(
		ctx.Metadata,
		ctx.Integration,
		ctx.HTTP,
		ctx.Configuration,
	)
}

func (c *CreateDeployment) Execute(ctx core.ExecutionContext) error {
	var config CreateDeploymentConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if err := validateCreateDeploymentConfig(config); err != nil {
		return err
	}

	client, err := common.NewClient(ctx.Integration, ctx.HTTP)
	if err != nil {
		return fmt.Errorf("failed to initialize GitHub client: %w", err)
	}

	req := newGitHubDeploymentRequest(config)

	callCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	deployment, err := createDeploymentWithRetry(callCtx, client, config.Repository, req)
	if err != nil {
		return fmt.Errorf("failed to create deployment: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"github.deployment",
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

const maxCreateDeploymentAttempts = 8

// createDeploymentWithRetry calls the Deployments API and retries when GitHub returns
// HTTP 202 Accepted (go-github reports this as *github.AcceptedError), which happens
// while GitHub schedules work to evaluate required_contexts against commit statuses.
func createDeploymentWithRetry(ctx context.Context, client *common.Client, repository string, req *github.DeploymentRequest) (*github.Deployment, error) {
	var lastAccepted *github.AcceptedError
	for attempt := 0; attempt < maxCreateDeploymentAttempts; attempt++ {
		if attempt > 0 {
			delaySec := 1 << uint(attempt-1)
			if delaySec > 16 {
				delaySec = 16
			}
			select {
			case <-ctx.Done():
				if lastAccepted != nil {
					return nil, fmt.Errorf("context done while retrying deployment after 202 Accepted: %w", ctx.Err())
				}
				return nil, ctx.Err()
			case <-time.After(time.Duration(delaySec) * time.Second):
			}
		}

		deployment, _, err := client.CreateDeployment(ctx, repository, req)
		if err == nil {
			return deployment, nil
		}

		var accepted *github.AcceptedError
		if errors.As(err, &accepted) {
			lastAccepted = accepted
			continue
		}

		return nil, err
	}

	if lastAccepted != nil {
		return nil, fmt.Errorf("%d consecutive 202 Accepted responses from GitHub (deployment still being prepared; wait and re-run, or leave required contexts empty for previews): %w", maxCreateDeploymentAttempts, lastAccepted)
	}

	return nil, fmt.Errorf("create deployment failed after %d attempts", maxCreateDeploymentAttempts)
}

func validateCreateDeploymentConfig(config CreateDeploymentConfiguration) error {
	if strings.TrimSpace(config.Ref) == "" {
		return fmt.Errorf("ref is required")
	}
	if strings.TrimSpace(config.Environment) == "" {
		return fmt.Errorf("environment is required")
	}
	return nil
}

func newGitHubDeploymentRequest(config CreateDeploymentConfiguration) *github.DeploymentRequest {
	ref := strings.TrimSpace(config.Ref)
	env := strings.TrimSpace(config.Environment)

	req := &github.DeploymentRequest{
		Ref:                   github.Ptr(ref),
		Environment:           github.Ptr(env),
		AutoMerge:             github.Ptr(config.AutoMerge),
		TransientEnvironment:  github.Ptr(config.TransientEnvironment),
		ProductionEnvironment: github.Ptr(config.ProductionEnvironment),
	}

	if strings.TrimSpace(config.Description) != "" {
		req.Description = github.Ptr(strings.TrimSpace(config.Description))
	}
	if strings.TrimSpace(config.Task) != "" {
		req.Task = github.Ptr(strings.TrimSpace(config.Task))
	}

	applyRequiredContextsToDeploymentRequest(req, config)

	return req
}

// normalizeRequiredContexts trims entries, drops blanks, and de-duplicates while preserving order.
func normalizeRequiredContexts(raw []string) []string {
	seen := make(map[string]struct{}, len(raw))
	out := make([]string, 0, len(raw))
	for _, s := range raw {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

func applyRequiredContextsToDeploymentRequest(req *github.DeploymentRequest, config CreateDeploymentConfiguration) {
	contexts := normalizeRequiredContexts(config.RequiredContexts)
	if len(contexts) > 0 {
		c := append([]string(nil), contexts...)
		req.RequiredContexts = &c
		return
	}
	//
	// Empty required_contexts bypasses GitHub's default (which waits for every context).
	//
	empty := []string{}
	req.RequiredContexts = &empty
}
