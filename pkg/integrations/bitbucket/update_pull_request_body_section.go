package bitbucket

import (
	"context"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/prbody"
)

type UpdatePullRequestBodySection struct{ prbody.Action }

func (*UpdatePullRequestBodySection) ExampleOutput() map[string]any { return prbody.ExampleOutput() }

func (*UpdatePullRequestBodySection) Name() string { return "bitbucket.updatePullRequestBodySection" }
func (*UpdatePullRequestBodySection) Description() string {
	return "Update one Markdown section in a Bitbucket pull request description"
}
func (*UpdatePullRequestBodySection) Icon() string  { return "bitbucket" }
func (*UpdatePullRequestBodySection) Color() string { return "blue" }
func (c *UpdatePullRequestBodySection) Setup(ctx core.SetupContext) error {
	config, err := prbody.Decode(ctx.Configuration)
	if err != nil {
		return err
	}
	if err := config.Validate(); err != nil {
		return err
	}
	return setupRepository(ctx, config.Repository)
}
func (*UpdatePullRequestBodySection) HandleHook(ctx core.ActionHookContext) error {
	config, err := prbody.Decode(ctx.Configuration)
	if err != nil {
		return err
	}
	client, err := NewBodySectionClient(ctx.Integration, ctx.HTTP, config.Repository)
	if err != nil {
		return err
	}
	return prbody.HandleHook(ctx, client)
}

type bodySectionClient struct{ client *Client }

func NewBodySectionClient(integration core.IntegrationContext, http core.HTTPContext, repository string) (prbody.Client, error) {
	if err := requireRepositoryInWorkspace(integration, repository); err != nil {
		return nil, err
	}
	client, err := newIntegrationClient(http, integration)
	if err != nil {
		return nil, err
	}
	return &bodySectionClient{client: client}, nil
}
func (c *bodySectionClient) Read(_ context.Context, repository string, number int64) (prbody.Description, error) {
	pr, err := c.client.GetPullRequest(repository, number)
	if err != nil {
		return prbody.Description{}, err
	}
	return prbody.Description{Body: pr.Description, Open: pr.State == PullRequestStateOpen, HeadSHA: pr.SourceHash}, nil
}
func (c *bodySectionClient) Write(_ context.Context, repository string, number int64, body string) error {
	_, err := c.client.UpdatePullRequest(repository, number, UpdatePullRequestRequest{Description: &body})
	return err
}
