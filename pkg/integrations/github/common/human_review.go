package common

import (
	"context"

	"github.com/google/go-github/v84/github"
)

// Human review uses the installation client. The agent cannot publish this check.
func (c *Client) CreateHumanReviewCheck(ctx context.Context, repository string, options github.CreateCheckRunOptions) (*github.CheckRun, *github.Response, error) {
	return c.CreateCheckRun(ctx, repository, options)
}

func (c *Client) UpdateHumanReviewCheck(ctx context.Context, repository string, id int64, options github.UpdateCheckRunOptions) (*github.CheckRun, *github.Response, error) {
	return c.UpdateCheckRun(ctx, repository, id, options)
}

func (c *Client) HumanReviewTeamMembers(ctx context.Context, organization, slug string) ([]*github.User, error) {
	options := &github.TeamListTeamMembersOptions{ListOptions: github.ListOptions{PerPage: 100}}
	var members []*github.User
	for {
		page, response, err := c.underlying.Teams.ListTeamMembersBySlug(ctx, organization, slug, options)
		if err != nil {
			return nil, err
		}
		members = append(members, page...)
		if response == nil || response.NextPage == 0 {
			return members, nil
		}
		options.Page = response.NextPage
	}
}

func (c *Client) HumanReviewChangedFiles(ctx context.Context, repository string, number int) ([]*github.CommitFile, error) {
	owner, name := c.ownerAndName(repository)
	options := &github.ListOptions{PerPage: 100}
	var files []*github.CommitFile
	for {
		page, response, err := c.underlying.PullRequests.ListFiles(ctx, owner, name, number, options)
		if err != nil {
			return nil, err
		}
		files = append(files, page...)
		if response == nil || response.NextPage == 0 {
			return files, nil
		}
		options.Page = response.NextPage
	}
}

func (c *Client) HumanReviewOwnershipFile(ctx context.Context, repository, path, ref string) (string, error) {
	owner, name := c.ownerAndName(repository)
	file, _, _, err := c.underlying.Repositories.GetContents(ctx, owner, name, path, &github.RepositoryContentGetOptions{Ref: ref})
	if IsNotFoundError(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if file == nil {
		return "", nil
	}
	return file.GetContent()
}

func (c *Client) GetTeamBySlug(ctx context.Context, organization, slug string) (*github.Team, *github.Response, error) {
	return c.underlying.Teams.GetTeamBySlug(ctx, organization, slug)
}
