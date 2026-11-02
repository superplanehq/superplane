package linear

import (
	"fmt"
	"slices"
	"strings"
)

// IssueScope describes which issues a factory intake accepts.
//
// A scope with a TeamID is team mode. It accepts every issue of the team.
// ProjectIDs narrows it to those projects, and IncludeNoProject keeps issues
// that have no project in that narrowed set.
//
// A scope without a TeamID is legacy mode, for intakes that only know their
// projects. It accepts the issues of ProjectIDs and never an issue without a
// project, because the issue could belong to any team of the workspace.
type IssueScope struct {
	TeamID           string
	ProjectIDs       []string
	IncludeNoProject bool
}

// Validate reports a scope that cannot select any issue.
func (s IssueScope) Validate() error {
	if strings.TrimSpace(s.TeamID) == "" && len(normalizeIDs(s.ProjectIDs)) == 0 {
		return fmt.Errorf("a team or at least one project is required")
	}
	return nil
}

// Contains reports whether the issue is inside the scope. Issue state and
// labels are not part of the scope.
func (s IssueScope) Contains(issue Issue) bool {
	teamID := strings.TrimSpace(s.TeamID)
	projectIDs := normalizeIDs(s.ProjectIDs)
	if teamID == "" && len(projectIDs) == 0 {
		return false
	}
	if teamID != "" && (issue.Team == nil || issue.Team.ID != teamID) {
		return false
	}
	if len(projectIDs) == 0 {
		return true
	}
	if issue.Project == nil {
		return s.acceptsNoProject()
	}
	return slices.Contains(projectIDs, issue.Project.ID)
}

// acceptsNoProject only applies to a team scope with a project filter.
// Without a project filter every issue of the team is accepted anyway, and
// without a team the issue has no known owner.
func (s IssueScope) acceptsNoProject() bool {
	return s.IncludeNoProject && strings.TrimSpace(s.TeamID) != ""
}

// issueFilter builds the Linear IssueFilter for open issues in the scope,
// optionally narrowed by a title or description text search.
func (s IssueScope) issueFilter(text string) map[string]any {
	filter := map[string]any{
		"state": map[string]any{"type": map[string]any{"nin": []string{"completed", "canceled"}}},
	}
	if teamID := strings.TrimSpace(s.TeamID); teamID != "" {
		filter["team"] = map[string]any{"id": map[string]any{"eq": teamID}}
	}

	// A filter object holds one "or" key. When the project rule and the text
	// search both need one, each moves into its own entry under "and".
	alternatives := [][]map[string]any{}
	if projectIDs := normalizeIDs(s.ProjectIDs); len(projectIDs) > 0 {
		inProjects := map[string]any{"id": map[string]any{"in": projectIDs}}
		if s.acceptsNoProject() {
			alternatives = append(alternatives, []map[string]any{
				{"project": inProjects},
				{"project": map[string]any{"null": true}},
			})
		} else {
			filter["project"] = inProjects
		}
	}
	if text != "" {
		alternatives = append(alternatives, []map[string]any{
			{"title": map[string]any{"containsIgnoreCase": text}},
			{"description": map[string]any{"containsIgnoreCase": text}},
		})
	}

	switch len(alternatives) {
	case 0:
	case 1:
		filter["or"] = alternatives[0]
	default:
		conjunction := make([]map[string]any, 0, len(alternatives))
		for _, alternative := range alternatives {
			conjunction = append(conjunction, map[string]any{"or": alternative})
		}
		filter["and"] = conjunction
	}
	return filter
}
