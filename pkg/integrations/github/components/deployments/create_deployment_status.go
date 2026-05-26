package deployments

import (
	"context"
	"fmt"
	"math"
	"net/url"
	"strconv"
	"strings"

	"github.com/google/go-github/v84/github"
	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
)

type CreateDeploymentStatus struct{}

type CreateDeploymentStatusConfiguration struct {
	Repository     string `json:"repository" mapstructure:"repository"`
	DeploymentID   string `json:"deploymentId" mapstructure:"deploymentId"`
	State          string `json:"state" mapstructure:"state"`
	Description    string `json:"description" mapstructure:"description"`
	EnvironmentURL string `json:"environmentUrl" mapstructure:"environmentUrl"`
	LogURL         string `json:"logUrl" mapstructure:"logUrl"`
	Environment    string `json:"environment" mapstructure:"environment"`
}

func (c *CreateDeploymentStatus) Name() string {
	return "github.createDeploymentStatus"
}

func (c *CreateDeploymentStatus) Label() string {
	return "Create Deployment Status"
}

func (c *CreateDeploymentStatus) Description() string {
	return "Update the status of a GitHub deployment (success, failure, inactive, etc.)"
}

func (c *CreateDeploymentStatus) Documentation() string {
	return `The Create Deployment Status component posts a new status for an existing deployment. Use it after provisioning succeeds or fails, or to mark a preview as **inactive** when tearing down.

## Use Cases

- **Preview ready**: success with **Environment URL** for the **View deployment** link
- **Failed build or deploy**: failure or error with **Log URL**
- **Teardown**: inactive when the environment is removed

## Configuration

- **Repository**: GitHub repository
- **Deployment ID**: Numeric deployment id (often from a prior **Create Deployment** step)
- **State**: pending, queued, in_progress, success, failure, error, or inactive
- **Description**: Optional status description
- **Environment URL**: Live preview URL for **View deployment** (GitHub requires http(s); host-only values get https:// prepended)
- **Log URL**: Link to logs (same http(s) rules; GitHub validates this like target_url)
- **Environment**: Optional environment label override on the status

## Output

Emits github.deploymentStatus with the created status record.`
}

func (c *CreateDeploymentStatus) Icon() string {
	return "github"
}

func (c *CreateDeploymentStatus) Color() string {
	return "gray"
}

func (c *CreateDeploymentStatus) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateDeploymentStatus) Configuration() []configuration.Field {
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
			Name:        "deploymentId",
			Label:       "Deployment ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "e.g. {{ $['Create Deployment'].data.id }}",
			Description: "GitHub deployment id (integer)",
		},
		{
			Name:     "state",
			Label:    "State",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Pending", Value: "pending"},
						{Label: "Queued", Value: "queued"},
						{Label: "In progress", Value: "in_progress"},
						{Label: "Success", Value: "success"},
						{Label: "Failure", Value: "failure"},
						{Label: "Error", Value: "error"},
						{Label: "Inactive", Value: "inactive"},
					},
				},
			},
		},
		{
			Name:        "description",
			Label:       "Description",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Placeholder: "Optional status description",
		},
		{
			Name:        "environmentUrl",
			Label:       "Environment URL",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Placeholder: "https://preview.example.com",
			Description: "http(s) required by GitHub; scheme omitted → https:// is added",
		},
		{
			Name:        "logUrl",
			Label:       "Log URL",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Placeholder: "https://...",
			Description: "http(s) if set; scheme omitted → https:// is added",
		},
		{
			Name:        "environment",
			Label:       "Environment",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Placeholder: "Optional override for the status environment label",
		},
	}
}

func (c *CreateDeploymentStatus) Setup(ctx core.SetupContext) error {
	return common.EnsureRepoInMetadata(
		ctx.Metadata,
		ctx.Integration,
		ctx.HTTP,
		ctx.Configuration,
	)
}

func parseDeploymentID(raw string) (int64, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return 0, fmt.Errorf("deployment id must be a positive integer, got %q", raw)
	}

	if id, err := strconv.ParseInt(s, 10, 64); err == nil && id >= 1 {
		return id, nil
	}

	//
	// Expressions sometimes coerce large GitHub IDs through JSON numbers, which
	// stringify as scientific notation (e.g. 4.671220334e+09). ParseFloat accepts that.
	//
	f, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsNaN(f) || math.IsInf(f, 0) {
		return 0, fmt.Errorf("deployment id must be a positive integer, got %q", raw)
	}
	if f < 1 || math.Trunc(f) != f {
		return 0, fmt.Errorf("deployment id must be a positive integer, got %q", raw)
	}
	if f > float64(math.MaxInt64) {
		return 0, fmt.Errorf("deployment id must be a positive integer, got %q", raw)
	}

	id := int64(f)
	if id < 1 {
		return 0, fmt.Errorf("deployment id must be a positive integer, got %q", raw)
	}

	return id, nil
}

// normalizeGitHubDeploymentsAPIURL ensures GitHub's deployment status API accepts the URL:
// environment_url and log_url must use http or https. Values without a scheme get https:// prepended.
func normalizeGitHubDeploymentsAPIURL(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", fmt.Errorf("URL is empty")
	}
	if !strings.Contains(s, "://") {
		s = "https://" + s
	}
	u, err := url.Parse(s)
	if err != nil {
		return "", fmt.Errorf("invalid URL %q: %w", raw, err)
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
	default:
		return "", fmt.Errorf("URL %q must use http or https scheme (GitHub API requirement)", raw)
	}
	if u.Hostname() == "" {
		return "", fmt.Errorf("URL %q must include a hostname", raw)
	}
	return u.String(), nil
}

func (c *CreateDeploymentStatus) Execute(ctx core.ExecutionContext) error {
	var config CreateDeploymentStatusConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	deploymentID, err := parseDeploymentID(config.DeploymentID)
	if err != nil {
		return err
	}

	client, err := common.NewClient(ctx.Integration, ctx.HTTP)
	if err != nil {
		return fmt.Errorf("failed to initialize GitHub client: %w", err)
	}

	req := &github.DeploymentStatusRequest{
		State: github.Ptr(config.State),
	}

	if strings.TrimSpace(config.Description) != "" {
		req.Description = github.Ptr(strings.TrimSpace(config.Description))
	}
	if strings.TrimSpace(config.EnvironmentURL) != "" {
		envURL, err := normalizeGitHubDeploymentsAPIURL(config.EnvironmentURL)
		if err != nil {
			return fmt.Errorf("environment URL: %w", err)
		}
		req.EnvironmentURL = github.Ptr(envURL)
	}
	if strings.TrimSpace(config.LogURL) != "" {
		logURL, err := normalizeGitHubDeploymentsAPIURL(config.LogURL)
		if err != nil {
			return fmt.Errorf("log URL: %w", err)
		}
		req.LogURL = github.Ptr(logURL)
	}
	if strings.TrimSpace(config.Environment) != "" {
		req.Environment = github.Ptr(strings.TrimSpace(config.Environment))
	}

	status, _, err := client.CreateDeploymentStatus(context.Background(), config.Repository, deploymentID, req)
	if err != nil {
		return fmt.Errorf("failed to create deployment status: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"github.deploymentStatus",
		[]any{status},
	)
}

func (c *CreateDeploymentStatus) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateDeploymentStatus) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}

func (c *CreateDeploymentStatus) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateDeploymentStatus) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *CreateDeploymentStatus) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *CreateDeploymentStatus) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
