package linear

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

// WorkspaceProject is a Linear project the token can see, with the teams
// that own it. A project can belong to more than one team, including a
// private team the authorizing user is a member of.
type WorkspaceProject struct {
	ID      string
	Name    string
	TeamIDs []string
}

type workspaceProjectNode struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Teams struct {
		Nodes []struct {
			ID string `json:"id"`
		} `json:"nodes"`
	} `json:"teams"`
}

func (n workspaceProjectNode) project() WorkspaceProject {
	teamIDs := make([]string, 0, len(n.Teams.Nodes))
	for _, team := range n.Teams.Nodes {
		id := strings.TrimSpace(team.ID)
		if id == "" || slices.Contains(teamIDs, id) {
			continue
		}
		teamIDs = append(teamIDs, id)
	}
	slices.Sort(teamIDs)
	return WorkspaceProject{
		ID:      n.ID,
		Name:    n.Name,
		TeamIDs: teamIDs,
	}
}

const workspaceProjectsQuery = `
query WorkspaceProjects($first: Int!, $after: String) {
  projects(first: $first, after: $after) {
    nodes {
      id
      name
      teams(first: 50) { nodes { id } }
    }
    pageInfo { hasNextPage endCursor }
  }
}`

// ListWorkspaceProjects returns every project the token can read, including
// projects on private teams the authorizing user belongs to.
func (c *Client) ListWorkspaceProjects() ([]WorkspaceProject, error) {
	nodes, err := collectPages(c, workspaceProjectsQuery, map[string]any{"first": pageSize}, decodeWorkspaceProjects)
	if err != nil {
		return nil, err
	}
	return workspaceProjectsFromNodes(nodes), nil
}

const projectsByIDQuery = `
query ProjectsByID($ids: [ID!]!, $first: Int!, $after: String) {
  projects(first: $first, after: $after, filter: { id: { in: $ids } }) {
    nodes {
      id
      name
      teams(first: 50) { nodes { id } }
    }
    pageInfo { hasNextPage endCursor }
  }
}`

// ProjectTeamIDs returns the teams that own the given projects. The result
// is sorted and has no duplicates. Every requested project must be readable.
// A project the token cannot read is an error, so setup does not register
// webhooks for only part of the intake.
func (c *Client) ProjectTeamIDs(projectIDs []string) ([]string, error) {
	ids := normalizeIDs(projectIDs)
	if len(ids) == 0 {
		return nil, fmt.Errorf("at least one project is required")
	}

	nodes, err := collectPages(c, projectsByIDQuery, map[string]any{
		"ids":   ids,
		"first": pageSize,
	}, decodeWorkspaceProjects)
	if err != nil {
		return nil, err
	}

	found := map[string]struct{}{}
	teamIDs := []string{}
	for _, project := range workspaceProjectsFromNodes(nodes) {
		id := strings.TrimSpace(project.ID)
		if id == "" {
			continue
		}
		found[id] = struct{}{}
		for _, teamID := range project.TeamIDs {
			if !slices.Contains(teamIDs, teamID) {
				teamIDs = append(teamIDs, teamID)
			}
		}
	}
	for _, id := range ids {
		if _, ok := found[id]; !ok {
			return nil, fmt.Errorf("selected projects were not found")
		}
	}
	if len(teamIDs) == 0 {
		return nil, fmt.Errorf("selected projects were not found")
	}
	slices.Sort(teamIDs)
	return teamIDs, nil
}

func decodeWorkspaceProjects(data json.RawMessage) (*connection[workspaceProjectNode], error) {
	response := struct {
		Projects connection[workspaceProjectNode] `json:"projects"`
	}{}
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("error parsing projects: %v", err)
	}
	return &response.Projects, nil
}

func workspaceProjectsFromNodes(nodes []workspaceProjectNode) []WorkspaceProject {
	projects := make([]WorkspaceProject, 0, len(nodes))
	for _, node := range nodes {
		if strings.TrimSpace(node.ID) == "" {
			continue
		}
		projects = append(projects, node.project())
	}
	return projects
}

const openProjectIssuesQuery = `
query OpenProjectIssues($ids: [ID!]!, $first: Int!) {
  issues(
    first: $first
    orderBy: updatedAt
    filter: {
      project: { id: { in: $ids } }
      state: { type: { nin: ["completed", "canceled"] } }
    }
  ) {
    nodes {` + issueFields + `
    }
  }
}`

const searchProjectIssuesQuery = `
query SearchProjectIssues($ids: [ID!]!, $query: String!, $first: Int!) {
  issues(
    first: $first
    orderBy: updatedAt
    filter: {
      project: { id: { in: $ids } }
      state: { type: { nin: ["completed", "canceled"] } }
      or: [
        { title: { containsIgnoreCase: $query } }
        { description: { containsIgnoreCase: $query } }
      ]
    }
  ) {
    nodes {` + issueFields + `
    }
  }
}`

// ListOpenProjectIssues returns the newest open issues in the given projects.
// Completed and canceled issues are left out. limit is capped at one page.
func (c *Client) ListOpenProjectIssues(projectIDs []string, limit int) ([]Issue, error) {
	return c.SearchOpenProjectIssues(projectIDs, "", limit)
}

// SearchOpenProjectIssues returns open issues in the given projects whose
// title or description contains query. An empty query lists the newest open
// issues. An identifier such as ENG-142 is included when that issue belongs
// to one of the projects.
func (c *Client) SearchOpenProjectIssues(projectIDs []string, query string, limit int) ([]Issue, error) {
	ids := normalizeIDs(projectIDs)
	if len(ids) == 0 {
		return nil, fmt.Errorf("at least one project is required")
	}
	if limit <= 0 {
		limit = 10
	}
	if limit > pageSize {
		limit = pageSize
	}

	query = strings.TrimSpace(query)
	document := openProjectIssuesQuery
	variables := map[string]any{"ids": ids, "first": limit}
	if query != "" {
		document = searchProjectIssuesQuery
		variables["query"] = query
	}

	response := struct {
		Issues connection[Issue] `json:"issues"`
	}{}
	if err := c.execute(document, variables, &response); err != nil {
		return nil, err
	}
	issues := response.Issues.Nodes
	if query == "" || !looksLikeIssueIdentifier(query) {
		return issues, nil
	}

	issue, err := c.GetIssue(query)
	if err != nil || issue == nil || !issueInProjects(*issue, ids) || issueIsClosed(*issue) {
		return issues, nil
	}
	if slices.ContainsFunc(issues, func(existing Issue) bool { return existing.ID == issue.ID }) {
		return issues, nil
	}
	return append([]Issue{*issue}, issues...), nil
}

func normalizeIDs(ids []string) []string {
	normalized := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" || slices.Contains(normalized, id) {
			continue
		}
		normalized = append(normalized, id)
	}
	return normalized
}

func looksLikeIssueIdentifier(query string) bool {
	team, number, ok := strings.Cut(query, "-")
	if !ok || team == "" || number == "" {
		return false
	}
	for _, char := range number {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}

func issueInProjects(issue Issue, projectIDs []string) bool {
	if issue.Project == nil {
		return false
	}
	return slices.Contains(projectIDs, issue.Project.ID)
}

func issueIsClosed(issue Issue) bool {
	if issue.State == nil {
		return false
	}
	return issue.State.Type == "completed" || issue.State.Type == "canceled"
}
