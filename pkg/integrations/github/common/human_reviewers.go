package common

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-github/v84/github"
)

// ListHumanReviewers returns individual humans with write-or-higher access.
// An incomplete lookup is an error, so selection cannot treat it as no match.
func (c *Client) ListHumanReviewers(ctx context.Context, repository string) ([]*github.User, error) {
	owner, name := c.ownerAndName(repository)
	options := &github.ListCollaboratorsOptions{Permission: "push", ListOptions: github.ListOptions{PerPage: 100}}
	users := []*github.User{}
	seen := map[string]bool{}
	for {
		page, response, err := c.underlying.Repositories.ListCollaborators(ctx, owner, name, options)
		if err != nil {
			return nil, fmt.Errorf("list human reviewers for %s: %w", repository, err)
		}
		for _, user := range page {
			if user == nil || !strings.EqualFold(user.GetType(), "User") {
				continue
			}
			login := strings.ToLower(strings.TrimSpace(user.GetLogin()))
			if login == "" || strings.HasSuffix(login, "[bot]") || seen[login] {
				continue
			}
			if !user.GetPermissions().GetPush() && !user.GetPermissions().GetMaintain() && !user.GetPermissions().GetAdmin() {
				continue
			}
			users = append(users, user)
			seen[login] = true
		}
		if response == nil || response.NextPage == 0 {
			return users, nil
		}
		options.Page = response.NextPage
	}
}
