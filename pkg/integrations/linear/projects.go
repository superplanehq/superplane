package linear

import (
	"encoding/json"
	"fmt"
	"regexp"
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
			ID  string `json:"id"`
			Key string `json:"key"`
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
      teams(first: 50) { nodes { id key } }
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
      teams(first: 50) { nodes { id key } }
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

// ProjectIssueSearchResult includes exact identifier matches even outside the
// selected projects or open states, so callers can explain intake restrictions.
type ProjectIssueSearchResult struct {
	Issue
	ExactMatch bool
	// ExplicitIdentifier excludes identifiers inferred from a bare number.
	ExplicitIdentifier bool
}

// SearchOpenProjectIssues returns open issues in the selected projects.
// Complete identifiers (including ENG-12) mean an exact issue number, not a
// numeric prefix. Incomplete identifiers such as ENG- use text search only.
func (c *Client) SearchOpenProjectIssues(projectIDs []string, query string, limit int) ([]Issue, error) {
	results, err := c.SearchProjectIssues(projectIDs, query, limit)
	if err != nil {
		return nil, err
	}
	issues := make([]Issue, 0, len(results))
	for _, result := range results {
		if !result.ExactMatch || (issueInProjects(result.Issue, normalizeIDs(projectIDs)) && !issueIsClosed(result.Issue)) {
			issues = append(issues, result.Issue)
		}
	}
	return issues, nil
}

// SearchProjectIssues combines scoped title/description search with direct
// identifier lookup. Exact matches come first, including issues outside scope.
// Bare-number exact matches do not take a result slot. Callers drop the ones
// that fail intake checks, and a short page hides further matches.
func (c *Client) SearchProjectIssues(projectIDs []string, query string, limit int) ([]ProjectIssueSearchResult, error) {
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
	identifiers, err := c.projectIssueIdentifiers(ids, query)
	if err != nil {
		return nil, err
	}
	exactMatches := make([]ProjectIssueSearchResult, 0, limit)
	seen := map[string]bool{}
	explicitIdentifier := !issueNumberPattern.MatchString(query)
	if len(identifiers) > 0 {
		exact := struct {
			Issues connection[Issue] `json:"issues"`
		}{}
		if err := c.execute(identifierIssuesQuery, map[string]any{"identifiers": identifiers, "first": limit}, &exact); err != nil {
			return nil, err
		}
		for _, issue := range exact.Issues.Nodes {
			if seen[issue.ID] {
				continue
			}
			exactMatches = append(exactMatches, ProjectIssueSearchResult{
				Issue:              issue,
				ExactMatch:         true,
				ExplicitIdentifier: explicitIdentifier,
			})
			seen[issue.ID] = true
		}
	}
	textMatches := make([]ProjectIssueSearchResult, 0, len(response.Issues.Nodes))
	for _, issue := range response.Issues.Nodes {
		if seen[issue.ID] {
			continue
		}
		textMatches = append(textMatches, ProjectIssueSearchResult{Issue: issue})
		seen[issue.ID] = true
	}
	return mergeProjectIssueSearchResults(exactMatches, textMatches, limit), nil
}

// Explicit identifier matches occupy result slots. Bare-number exact matches
// stay in the list without replacing title or description matches.
func mergeProjectIssueSearchResults(exactMatches, textMatches []ProjectIssueSearchResult, limit int) []ProjectIssueSearchResult {
	occupied := 0
	for _, result := range exactMatches {
		if result.ExplicitIdentifier {
			occupied++
		}
	}
	textRoom := limit - occupied
	if textRoom < 0 {
		textRoom = 0
	}
	if len(textMatches) > textRoom {
		textMatches = textMatches[:textRoom]
	}
	return append(exactMatches, textMatches...)
}

// Linear's ID comparator accepts human-readable identifiers as well as UUIDs.
// Keep this lookup unscoped so the intake can explain excluded exact matches.
const identifierIssuesQuery = `
query IdentifierIssues($identifiers: [ID!]!, $first: Int!) {
 issues(first: $first, filter: { id: { in: $identifiers } }) {
  nodes {` + issueFields + `}
 }
}`

var issueIdentifierPattern = regexp.MustCompile(`(?i)\b([a-z][a-z0-9]*)[- ]([0-9]+)\b`)
var issueNumberPattern = regexp.MustCompile(`^[0-9]+$`)

func (c *Client) projectIssueIdentifiers(projectIDs []string, query string) ([]string, error) {
	if ref, ok := IssueRefFromURL(query); ok {
		query = ref.Identifier
	}
	if match := issueIdentifierPattern.FindStringSubmatch(query); match != nil {
		return []string{strings.ToUpper(match[1]) + "-" + match[2]}, nil
	}
	if !issueNumberPattern.MatchString(query) {
		return nil, nil
	}
	// A bare number is ambiguous across teams. Try every team of the intake's
	// projects, including issues that have since moved outside those projects.
	nodes, err := collectPages(c, projectsByIDQuery, map[string]any{
		"ids": projectIDs, "first": pageSize,
	}, decodeWorkspaceProjects)
	if err != nil {
		return nil, err
	}
	identifiers := []string{}
	for _, project := range nodes {
		for _, team := range project.Teams.Nodes {
			key := strings.ToUpper(strings.TrimSpace(team.Key))
			if key != "" {
				identifiers = append(identifiers, key+"-"+query)
			}
		}
	}
	return normalizeIDs(identifiers), nil
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
