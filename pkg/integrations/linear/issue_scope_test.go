package linear

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test__IssueScope__Contains(t *testing.T) {
	inTeam := func(project *Project) Issue {
		return Issue{Team: &Team{ID: "t1"}, Project: project}
	}

	for _, tc := range []struct {
		name  string
		scope IssueScope
		issue Issue
		want  bool
	}{
		{"team accepts an issue without a project", IssueScope{TeamID: "t1"}, inTeam(nil), true},
		{"team accepts an issue in any project", IssueScope{TeamID: "t1"}, inTeam(&Project{ID: "p9"}), true},
		{"team rejects another team", IssueScope{TeamID: "t1"}, Issue{Team: &Team{ID: "t2"}}, false},
		{"team rejects an issue with no team", IssueScope{TeamID: "t1"}, Issue{}, false},
		{"project filter accepts a selected project", IssueScope{TeamID: "t1", ProjectIDs: []string{"p1"}}, inTeam(&Project{ID: "p1"}), true},
		{"project filter rejects another project", IssueScope{TeamID: "t1", ProjectIDs: []string{"p1"}}, inTeam(&Project{ID: "p2"}), false},
		{"project filter rejects no project by default", IssueScope{TeamID: "t1", ProjectIDs: []string{"p1"}}, inTeam(nil), false},
		{"project filter accepts no project when included", IssueScope{TeamID: "t1", ProjectIDs: []string{"p1"}, IncludeNoProject: true}, inTeam(nil), true},
		{"project filter still rejects another project when no project is included", IssueScope{TeamID: "t1", ProjectIDs: []string{"p1"}, IncludeNoProject: true}, inTeam(&Project{ID: "p2"}), false},
		{"legacy accepts a selected project", IssueScope{ProjectIDs: []string{"p1"}}, Issue{Project: &Project{ID: "p1"}}, true},
		{"legacy rejects an issue without a project", IssueScope{ProjectIDs: []string{"p1"}, IncludeNoProject: true}, Issue{Team: &Team{ID: "t1"}}, false},
		{"empty scope rejects everything", IssueScope{}, inTeam(nil), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.scope.Contains(tc.issue))
		})
	}
}
