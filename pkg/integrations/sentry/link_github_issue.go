package sentry

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type LinkGitHubIssue struct{}

type LinkGitHubIssueConfiguration struct {
	IssueID             string `json:"issueId" mapstructure:"issueId"`
	GitHubIntegrationID string `json:"githubIntegrationId" mapstructure:"githubIntegrationId"`
	Repo                string `json:"repo" mapstructure:"repo"`
	ExternalIssue       string `json:"externalIssue" mapstructure:"externalIssue"`
	Comment             string `json:"comment" mapstructure:"comment"`
}

type LinkGitHubIssueNodeMetadata struct {
	IssueTitle             string `json:"issueTitle,omitempty" mapstructure:"issueTitle"`
	GitHubIntegrationLabel string `json:"githubIntegrationLabel,omitempty" mapstructure:"githubIntegrationLabel"`
	ExternalIssueLabel     string `json:"externalIssueLabel,omitempty" mapstructure:"externalIssueLabel"`
}

func (c *LinkGitHubIssue) Name() string {
	return "sentry.linkGitHubIssue"
}

func (c *LinkGitHubIssue) Label() string {
	return "Link GitHub Issue"
}

func (c *LinkGitHubIssue) Description() string {
	return "Link an existing GitHub issue to a Sentry issue through the Sentry GitHub integration"
}

func (c *LinkGitHubIssue) Documentation() string {
	return `The Link GitHub Issue component connects a Sentry issue to an existing GitHub issue using the GitHub integration installed in your Sentry organization.

## Use Cases

- **Automated triage**: link a GitHub issue created by an upstream workflow to the triggering Sentry exception
- **Cross-tool traceability**: connect incident tickets to the Sentry issues that caused them
- **Release follow-up**: associate remediation tasks in GitHub with unresolved Sentry errors

## Prerequisites

- A GitHub integration must be installed in your Sentry organization (**Settings → Integrations → GitHub**)
- The Sentry personal token must include **Issue & Event → Read & Write**

## Configuration

- **Issue**: Select the Sentry issue to link
- **GitHub Integration**: Select the GitHub integration installed in Sentry
- **Repository**: GitHub repository in owner/repo format
- **GitHub Issue Number**: The number of the existing GitHub issue to link
- **Comment**: Optional comment added when the link is created

## Output

Returns the external issue link object created by Sentry, including the linked issue URL and display name.`
}

func (c *LinkGitHubIssue) Icon() string {
	return "bug"
}

func (c *LinkGitHubIssue) Color() string {
	return "gray"
}

func (c *LinkGitHubIssue) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *LinkGitHubIssue) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "issueId",
			Label:       "Issue",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Select the Sentry issue to link",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeIssue,
				},
			},
		},
		{
			Name:        "githubIntegrationId",
			Label:       "GitHub Integration",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Select the GitHub integration installed in your Sentry organization",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeGitHubIntegration,
				},
			},
		},
		{
			Name:        "repo",
			Label:       "Repository",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "GitHub repository in owner/repo format",
		},
		{
			Name:        "externalIssue",
			Label:       "GitHub Issue Number",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Number of the existing GitHub issue to link",
		},
		{
			Name:        "comment",
			Label:       "Comment",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Description: "Optional comment added when the link is created",
		},
	}
}

func (c *LinkGitHubIssue) Setup(ctx core.SetupContext) error {
	config, err := decodeLinkGitHubIssueConfiguration(ctx.Configuration)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}
	normalizeLinkGitHubIssueConfiguration(&config)

	if err := validateLinkGitHubIssueConfiguration(config); err != nil {
		return err
	}

	if isExpressionValue(config.IssueID) || isExpressionValue(config.GitHubIntegrationID) {
		return ctx.Metadata.Set(LinkGitHubIssueNodeMetadata{})
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create sentry client: %w", err)
	}

	nodeMetadata := LinkGitHubIssueNodeMetadata{
		ExternalIssueLabel: displayExternalIssueLabel(config.Repo, config.ExternalIssue),
	}

	issue, err := client.GetIssue(config.IssueID)
	if err != nil {
		return fmt.Errorf("failed to retrieve sentry issue: %w", err)
	}
	nodeMetadata.IssueTitle = displayIssueLabel(issue.ShortID, issue.Title)

	integrations, err := client.ListOrganizationIntegrations("github")
	if err != nil {
		return fmt.Errorf("failed to list sentry github integrations: %w", err)
	}

	for _, integration := range integrations {
		if integration.ID == config.GitHubIntegrationID {
			nodeMetadata.GitHubIntegrationLabel = integration.Name
			break
		}
	}

	return ctx.Metadata.Set(nodeMetadata)
}

func validateLinkGitHubIssueConfiguration(config LinkGitHubIssueConfiguration) error {
	if config.IssueID == "" {
		return errors.New("issueId is required")
	}

	if config.GitHubIntegrationID == "" {
		return errors.New("githubIntegrationId is required")
	}

	if config.Repo == "" {
		return errors.New("repo is required")
	}

	if config.ExternalIssue == "" {
		return errors.New("externalIssue is required")
	}

	if !strings.Contains(config.Repo, "/") {
		return errors.New("repo must be in owner/repo format")
	}

	return nil
}

func normalizeLinkGitHubIssueConfiguration(config *LinkGitHubIssueConfiguration) {
	if config == nil {
		return
	}

	config.IssueID = strings.TrimSpace(config.IssueID)
	config.GitHubIntegrationID = strings.TrimSpace(config.GitHubIntegrationID)
	config.Repo = strings.TrimSpace(config.Repo)
	config.ExternalIssue = strings.TrimSpace(config.ExternalIssue)
	config.Comment = strings.TrimSpace(config.Comment)
}

func (c *LinkGitHubIssue) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *LinkGitHubIssue) Execute(ctx core.ExecutionContext) error {
	config, err := decodeLinkGitHubIssueConfiguration(ctx.Configuration)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}
	normalizeLinkGitHubIssueConfiguration(&config)

	if err := validateLinkGitHubIssueConfiguration(config); err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create sentry client: %w", err)
	}

	link, err := client.LinkExternalIssue(config.IssueID, config.GitHubIntegrationID, LinkExternalIssueRequest{
		Repo:          config.Repo,
		ExternalIssue: parseExternalIssueValue(config.ExternalIssue),
		Comment:       config.Comment,
	})
	if err != nil {
		return fmt.Errorf("failed to link github issue to sentry issue: %w", err)
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, "sentry.externalIssue", []any{link})
}

func (c *LinkGitHubIssue) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *LinkGitHubIssue) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *LinkGitHubIssue) Cleanup(ctx core.SetupContext) error {
	return nil
}

func decodeLinkGitHubIssueConfiguration(input any) (LinkGitHubIssueConfiguration, error) {
	config := LinkGitHubIssueConfiguration{}

	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:           &config,
		TagName:          "mapstructure",
		WeaklyTypedInput: true,
	})
	if err != nil {
		return LinkGitHubIssueConfiguration{}, err
	}

	if err := decoder.Decode(input); err != nil {
		return LinkGitHubIssueConfiguration{}, err
	}

	return config, nil
}

func parseExternalIssueValue(value string) any {
	if issueNumber, err := strconv.Atoi(value); err == nil {
		return issueNumber
	}

	return value
}

func displayExternalIssueLabel(repo, externalIssue string) string {
	repo = strings.TrimSpace(repo)
	externalIssue = strings.TrimSpace(externalIssue)

	switch {
	case repo != "" && externalIssue != "":
		return fmt.Sprintf("%s#%s", repo, externalIssue)
	case externalIssue != "":
		return externalIssue
	default:
		return repo
	}
}

func (c *LinkGitHubIssue) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *LinkGitHubIssue) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
