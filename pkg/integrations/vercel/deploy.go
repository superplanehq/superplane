package vercel

import (
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type TriggerDeployment struct{}

type TriggerDeploymentConfiguration struct {
	Project string `json:"project" mapstructure:"project"`
	GitRef  string `json:"gitRef" mapstructure:"gitRef"`
	Target  string `json:"target" mapstructure:"target"`
}

func (c *TriggerDeployment) Name() string {
	return "vercel.deploy"
}

func (c *TriggerDeployment) Label() string {
	return "Deploy"
}

func (c *TriggerDeployment) Description() string {
	return "Start a new deployment for a Git-connected Vercel project"
}

func (c *TriggerDeployment) Documentation() string {
	return `The Deploy component starts a new deployment for a Vercel project and returns the queued deployment.

## Use Cases

- **Merge to deploy**: Start a production deploy after a merge or CI pass
- **Scheduled deploys**: Redeploy on a schedule or after content changes
- **Chained deploys**: Deploy project B when project A finishes

## How It Works

1. Looks up the selected project and reads its Git connection
2. Starts a deployment from the configured branch or commit via the [Vercel REST API](https://vercel.com/docs/rest-api)
3. Emits the queued deployment immediately

The component does not wait for the build to finish. Pair it with the **On Deployment** trigger or the **Get Deployment** component to follow the build state.

## Configuration

- **Project**: Required. Must be connected to GitHub, GitLab, or Bitbucket.
- **Branch / Commit**: Optional. Defaults to the production branch of the project.
- **Target**: ` + "`production`" + ` (default) or ` + "`preview`" + `.`
}

func (c *TriggerDeployment) Icon() string {
	return "rocket"
}

func (c *TriggerDeployment) Color() string {
	return "gray"
}

func (c *TriggerDeployment) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *TriggerDeployment) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "project",
			Label:    "Project",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "project",
				},
			},
			Description: "Vercel project to deploy",
		},
		{
			Name:        "gitRef",
			Label:       "Branch / Commit",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Branch or commit SHA to deploy. Defaults to the production branch of the project.",
		},
		{
			Name:     "target",
			Label:    "Target",
			Type:     configuration.FieldTypeSelect,
			Required: false,
			Default:  targetProduction,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Production", Value: targetProduction},
						{Label: "Preview", Value: targetPreview},
					},
				},
			},
		},
	}
}

func decodeTriggerDeploymentConfiguration(configuration any) (TriggerDeploymentConfiguration, error) {
	spec := TriggerDeploymentConfiguration{}
	if err := mapstructure.Decode(configuration, &spec); err != nil {
		return spec, fmt.Errorf("failed to decode configuration: %w", err)
	}

	spec.Project = strings.TrimSpace(spec.Project)
	spec.GitRef = strings.TrimSpace(spec.GitRef)
	spec.Target = strings.TrimSpace(strings.ToLower(spec.Target))

	if spec.Project == "" {
		return spec, fmt.Errorf("project is required")
	}

	if spec.Target == "" {
		spec.Target = targetProduction
	}

	if spec.Target != targetProduction && spec.Target != targetPreview {
		return spec, fmt.Errorf("target must be production or preview")
	}

	return spec, nil
}

// gitSourceFromProject builds the gitSource request field from the Git link of a Vercel project.
func gitSourceFromProject(project *Project, requestedRef string) (*GitSource, error) {
	if project == nil || project.Link == nil ||
		!slices.Contains(allowedGitTypes, strings.ToLower(project.Link.Type)) ||
		project.Link.Org == "" || project.Link.Repo == "" {
		return nil, fmt.Errorf(
			"project %s is not connected to a supported Git repository (GitHub, GitLab, or Bitbucket)",
			projectLabel(project),
		)
	}

	ref := strings.TrimSpace(requestedRef)
	if ref == "" {
		ref = strings.TrimSpace(project.Link.ProductionBranch)
	}
	if ref == "" {
		ref = "main"
	}

	return &GitSource{
		Type: strings.ToLower(project.Link.Type),
		Org:  project.Link.Org,
		Repo: project.Link.Repo,
		Ref:  ref,
	}, nil
}

func projectLabel(project *Project) string {
	if project == nil || project.Name == "" {
		return "(unknown)"
	}

	return project.Name
}

func (c *TriggerDeployment) Setup(ctx core.SetupContext) error {
	_, err := decodeTriggerDeploymentConfiguration(ctx.Configuration)
	return err
}

func (c *TriggerDeployment) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeTriggerDeploymentConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	project, err := client.GetProject(spec.Project)
	if err != nil {
		return fmt.Errorf("failed to fetch Vercel project: %w", err)
	}

	gitSource, err := gitSourceFromProject(project, spec.GitRef)
	if err != nil {
		return err
	}

	request := CreateDeploymentRequest{
		Name:      spec.Project,
		GitSource: gitSource,
	}
	if spec.Target == targetProduction {
		request.Target = &spec.Target
	}

	deployment, err := client.CreateDeployment(request)
	if err != nil {
		return fmt.Errorf("failed to start Vercel deployment: %w", err)
	}

	data := deploymentData(deployment)
	if readString(data["target"]) == "" {
		// Preview deployments have no target in the Vercel response; report
		// the configured target so downstream nodes see what was requested.
		data["target"] = spec.Target
	}
	data["ref"] = gitSource.Ref

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		TriggerDeploymentPayloadType,
		[]any{data},
	)
}

func (c *TriggerDeployment) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *TriggerDeployment) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *TriggerDeployment) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *TriggerDeployment) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *TriggerDeployment) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
