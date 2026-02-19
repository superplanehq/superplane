// Package buildkite implements the Buildkite integration for SuperPlane.
package buildkite

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegration("buildkite", &Buildkite{})
}

type Buildkite struct{}

type Configuration struct {
	Organization string `json:"organization"`
	APIToken     string `json:"apiToken"`
}

type Metadata struct {
	Organizations []string `json:"organizations"`
	SetupComplete bool     `json:"setupComplete"`
	OrgSlug       string   `json:"orgSlug"`
}

func extractOrgSlug(orgInput string) (string, error) {
	if orgInput == "" {
		return "", fmt.Errorf("organization input is empty")
	}

	urlPattern := regexp.MustCompile(`(?:https?://)?(?:www\.)?buildkite\.com/(?:organizations/)?([^/]+)`)
	if matches := urlPattern.FindStringSubmatch(strings.TrimSpace(orgInput)); len(matches) > 1 {
		return matches[1], nil
	}

	// Just the slug (validate it looks like a valid org slug)
	slugPattern := regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*[a-zA-Z0-9]$`)
	if slugPattern.MatchString(strings.TrimSpace(orgInput)) {
		return strings.TrimSpace(orgInput), nil
	}

	return "", fmt.Errorf("invalid organization format: %s. Expected format: https://buildkite.com/my-org or just 'my-org'", orgInput)
}

func (b *Buildkite) createTokenSetupAction(ctx core.SyncContext) {
	ctx.Integration.NewBrowserAction(core.BrowserAction{
		Description: "Generate API token for triggering builds. Required permissions: `read_organizations`, `read_user`, `read_pipelines`, `read_builds`, `write_builds`.",
		URL:         "https://buildkite.com/user/api-access-tokens",
		Method:      "GET",
	})
}

func (b *Buildkite) Name() string {
	return "buildkite"
}

func (b *Buildkite) Label() string {
	return "Buildkite"
}

func (b *Buildkite) Icon() string {
	return "workflow"
}

func (b *Buildkite) Description() string {
	return "Trigger and react to your Buildkite builds"
}

func (b *Buildkite) Instructions() string {
	return "To create new Buildkite API key, open [Personal Settings > API Access Tokens](https://buildkite.com/user/api-access-tokens/new)."
}

func (b *Buildkite) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "organization",
			Label:       "Organization URL",
			Type:        configuration.FieldTypeString,
			Description: "Buildkite organization URL (e.g. https://buildkite.com/my-org or just my-org)",
			Placeholder: "e.g. https://buildkite.com/my-org or my-org",
			Required:    true,
		},
		{
			Name:        "apiToken",
			Label:       "API Token",
			Type:        configuration.FieldTypeString,
			Sensitive:   true,
			Description: "Buildkite API token with permissions: read_organizations, read_user, read_pipelines, read_builds, write_builds",
			Placeholder: "e.g. bkua_...",
			Required:    true,
		},
	}
}

func (b *Buildkite) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (b *Buildkite) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("Failed to decode configuration: %v", err)
	}

	if config.Organization == "" {
		return fmt.Errorf("Organization is required")
	}

	orgSlug, err := extractOrgSlug(config.Organization)
	if err != nil {
		return fmt.Errorf("Invalid organization format: %v", err)
	}

	// Prompt user to create API token
	if config.APIToken == "" {
		b.createTokenSetupAction(ctx)
		return nil
	}

	// Update metadata to track setup completion
	metadata := Metadata{}
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata); err != nil {
		metadata = Metadata{}
	}

	metadata.OrgSlug = orgSlug
	metadata.SetupComplete = true
	ctx.Integration.SetMetadata(metadata)

	ctx.Integration.Ready()
	return nil
}

func (b *Buildkite) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("error creating client: %v", err)
	}

	switch resourceType {

	case "pipeline":
		orgConfig, err := ctx.Integration.GetConfig("organization")
		if err != nil {
			return nil, fmt.Errorf("failed to get organization from integration config: %w", err)
		}
		orgSlug, err := extractOrgSlug(string(orgConfig))
		if err != nil {
			return nil, fmt.Errorf("failed to extract organization slug: %w", err)
		}

		pipelines, err := client.ListPipelines(orgSlug)
		if err != nil {
			return nil, fmt.Errorf("error listing pipelines: %v", err)
		}

		resources := make([]core.IntegrationResource, len(pipelines))
		for i, pipeline := range pipelines {
			resources[i] = core.IntegrationResource{
				Type: "pipeline",
				ID:   pipeline.Slug,
				Name: pipeline.Name,
			}
		}
		return resources, nil

	default:
		return nil, fmt.Errorf("unsupported resource type: %s", resourceType)
	}
}

func (b *Buildkite) Actions() []core.Action {
	return []core.Action{}
}

func (b *Buildkite) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}

func (b *Buildkite) HandleRequest(ctx core.HTTPRequestContext) {
	ctx.Response.WriteHeader(http.StatusNotFound)
}

func (b *Buildkite) Components() []core.Component {
	return []core.Component{
		&TriggerBuild{},
	}
}

func (b *Buildkite) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnBuildFinished{},
	}
}
