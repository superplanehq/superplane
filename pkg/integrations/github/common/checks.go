package common

import (
	"context"

	"github.com/google/go-github/v84/github"
)

func (c *Client) CreateCheckRun(ctx context.Context, repository string, options github.CreateCheckRunOptions) (*github.CheckRun, *github.Response, error) {
	owner, name := c.ownerAndName(repository)
	return c.underlying.Checks.CreateCheckRun(ctx, owner, name, options)
}

func (c *Client) UpdateCheckRun(ctx context.Context, repository string, id int64, options github.UpdateCheckRunOptions) (*github.CheckRun, *github.Response, error) {
	owner, name := c.ownerAndName(repository)
	return c.underlying.Checks.UpdateCheckRun(ctx, owner, name, id, options)
}
