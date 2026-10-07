package bitbucket

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const PayloadTypePullRequest = "bitbucket.pullRequest"

var expressionPlaceholderRegex = regexp.MustCompile(`(?s)\{\{.*?\}\}`)

func isExpression(value string) bool {
	return expressionPlaceholderRegex.MatchString(value)
}

// pullRequestAction holds the no-op parts of the Bitbucket pull request actions.
type pullRequestAction struct{}

func (pullRequestAction) Icon() string {
	return "bitbucket"
}

func (pullRequestAction) Color() string {
	return "blue"
}

func (pullRequestAction) Hooks() []core.Hook {
	return []core.Hook{}
}

func (pullRequestAction) HandleHook(ctx core.ActionHookContext) error {
	return nil
}

func (pullRequestAction) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (pullRequestAction) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (pullRequestAction) Cleanup(ctx core.SetupContext) error {
	return nil
}

func repositoryField() configuration.Field {
	return configuration.Field{
		Name:        "repository",
		Label:       "Repository",
		Type:        configuration.FieldTypeIntegrationResource,
		Required:    true,
		Description: "Repository in workspace/repository format.",
		TypeOptions: &configuration.TypeOptions{
			Resource: &configuration.ResourceTypeOptions{
				Type:           "repository",
				UseNameAsValue: true,
			},
		},
	}
}

func newIntegrationClient(httpCtx core.HTTPContext, integration core.IntegrationContext) (*Client, error) {
	metadata := Metadata{}
	if err := mapstructure.Decode(integration.GetMetadata(), &metadata); err != nil {
		return nil, fmt.Errorf("failed to decode integration metadata: %w", err)
	}
	return NewClient(metadata.AuthType, httpCtx, integration)
}

// setupRepository checks a literal repository against the workspace.
// Expressions resolve at run time, so Setup cannot check them.
func setupRepository(ctx core.SetupContext, repository string) error {
	repository = strings.TrimSpace(repository)
	if repository == "" {
		return fmt.Errorf("repository is required")
	}
	if isExpression(repository) {
		return nil
	}
	_, err := ensureRepoInMetadata(ctx.HTTP, ctx.Metadata, ctx.Integration, repository)
	return err
}

func parsePullRequestID(value string) (int64, error) {
	id, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("pull request ID must be a positive number: %q", value)
	}
	return id, nil
}
