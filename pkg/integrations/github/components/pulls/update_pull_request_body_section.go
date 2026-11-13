package pulls

import (
	"context"

	"github.com/google/go-github/v84/github"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/pkg/integrations/prbody"
)

type UpdatePullRequestBodySection struct{ prbody.Action }

func (*UpdatePullRequestBodySection) ExampleOutput() map[string]any { return prbody.ExampleOutput() }

func (*UpdatePullRequestBodySection) Name() string { return "github.updatePullRequestBodySection" }
func (*UpdatePullRequestBodySection) Description() string {
	return "Update one Markdown section in a GitHub pull request description"
}
func (*UpdatePullRequestBodySection) Icon() string  { return "github" }
func (*UpdatePullRequestBodySection) Color() string { return "gray" }
func (c *UpdatePullRequestBodySection) Setup(ctx core.SetupContext) error {
	config, err := prbody.Decode(ctx.Configuration)
	if err != nil {
		return err
	}
	if err := config.Validate(); err != nil {
		return err
	}
	return common.EnsureRepoInMetadata(ctx.Metadata, ctx.Integration, ctx.HTTP, ctx.Configuration)
}
func (*UpdatePullRequestBodySection) HandleHook(ctx core.ActionHookContext) error {
	client, err := NewBodySectionClient(ctx.Integration, ctx.HTTP)
	if err != nil {
		return err
	}
	return prbody.HandleHook(ctx, client)
}

type BodySectionClient struct{ client *common.Client }

func NewBodySectionClient(integration core.IntegrationContext, http core.HTTPContext) (prbody.Client, error) {
	client, err := common.NewClient(integration, http)
	if err != nil {
		return nil, err
	}
	return &BodySectionClient{client: client}, nil
}
func (c *BodySectionClient) GetCommit(ctx context.Context, repository, sha string) (*github.RepositoryCommit, error) {
	commit, _, err := c.client.GetCommit(ctx, repository, sha)
	return commit, err
}

func (c *BodySectionClient) Read(ctx context.Context, repository string, number int64) (prbody.Description, error) {
	pr, _, err := c.client.GetPullRequest(ctx, repository, int(number))
	if err != nil {
		return prbody.Description{}, err
	}
	return prbody.Description{Body: pr.GetBody(), Open: pr.GetState() == "open", HeadSHA: pr.GetHead().GetSHA()}, nil
}
func (c *BodySectionClient) Write(ctx context.Context, repository string, number int64, body string) error {
	_, _, err := c.client.EditPullRequest(ctx, repository, int(number), &github.PullRequest{Body: &body})
	return err
}
