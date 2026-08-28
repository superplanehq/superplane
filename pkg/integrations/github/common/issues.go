package common

import (
	"context"
	"strings"
	"time"

	"github.com/google/go-github/v84/github"
)

// maxIssuePageSize is the largest page GitHub GraphQL returns at once.
const maxIssuePageSize = 100

const newestOpenIssuesQuery = `query($owner: String!, $name: String!, $limit: Int!) {
  repository(owner: $owner, name: $name) {
    issues(first: $limit, states: OPEN, orderBy: {field: CREATED_AT, direction: DESC}) {
      nodes {
        id
        databaseId
        number
        title
        body
        url
        state
        createdAt
        updatedAt
        author { login }
        labels(first: 50) { nodes { name description color } }
        assignees(first: 50) { nodes { login databaseId } }
        comments { totalCount }
      }
    }
  }
}`

// ListNewestOpenIssues reads the newest open issues of a repository, newest
// first.
//
// It asks GraphQL instead of the REST issue list because REST answers that list
// with pull requests in the same page. A caller that wants N issues then has to
// read a larger page and drop the pull requests, and a repository with many
// open pull requests returns few issues, or none.
func (c *Client) ListNewestOpenIssues(ctx context.Context, repository string, limit int) ([]*github.Issue, error) {
	owner, name := c.ownerAndName(repository)

	var result issueQueryResult
	err := c.doGraphQL(ctx, newestOpenIssuesQuery, map[string]any{
		"owner": owner,
		"name":  name,
		"limit": min(max(limit, 1), maxIssuePageSize),
	}, &result)

	if err != nil {
		return nil, err
	}

	issues := make([]*github.Issue, 0, len(result.Repository.Issues.Nodes))
	for _, node := range result.Repository.Issues.Nodes {
		if node == nil {
			continue
		}

		issues = append(issues, node.issue())
	}

	return issues, nil
}

type issueQueryResult struct {
	Repository struct {
		Issues struct {
			Nodes []*issueNode `json:"nodes"`
		} `json:"issues"`
	} `json:"repository"`
}

type issueNode struct {
	NodeID     string    `json:"id"`
	DatabaseID int64     `json:"databaseId"`
	Number     int       `json:"number"`
	Title      string    `json:"title"`
	Body       string    `json:"body"`
	URL        string    `json:"url"`
	State      string    `json:"state"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`

	// GraphQL leaves the author out when the account is gone.
	Author *struct {
		Login string `json:"login"`
	} `json:"author"`

	Labels struct {
		Nodes []struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			Color       string `json:"color"`
		} `json:"nodes"`
	} `json:"labels"`

	Assignees struct {
		Nodes []struct {
			Login      string `json:"login"`
			DatabaseID int64  `json:"databaseId"`
		} `json:"nodes"`
	} `json:"assignees"`

	Comments struct {
		TotalCount int `json:"totalCount"`
	} `json:"comments"`
}

// issue shapes a GraphQL node like the REST body of the same issue. Callers
// and webhook handlers then read one issue format, whatever API delivered it.
func (n *issueNode) issue() *github.Issue {
	issue := &github.Issue{
		ID:        github.Ptr(n.DatabaseID),
		NodeID:    github.Ptr(n.NodeID),
		Number:    github.Ptr(n.Number),
		Title:     github.Ptr(n.Title),
		Body:      github.Ptr(n.Body),
		HTMLURL:   github.Ptr(n.URL),
		State:     github.Ptr(strings.ToLower(n.State)),
		Comments:  github.Ptr(n.Comments.TotalCount),
		CreatedAt: &github.Timestamp{Time: n.CreatedAt},
		UpdatedAt: &github.Timestamp{Time: n.UpdatedAt},
	}

	if n.Author != nil {
		issue.User = &github.User{Login: github.Ptr(n.Author.Login)}
	}

	for _, label := range n.Labels.Nodes {
		issue.Labels = append(issue.Labels, &github.Label{
			Name:        github.Ptr(label.Name),
			Description: github.Ptr(label.Description),
			Color:       github.Ptr(label.Color),
		})
	}

	for _, assignee := range n.Assignees.Nodes {
		issue.Assignees = append(issue.Assignees, &github.User{
			ID:    github.Ptr(assignee.DatabaseID),
			Login: github.Ptr(assignee.Login),
		})
	}

	return issue
}
