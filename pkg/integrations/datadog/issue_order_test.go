package datadog

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewestErrorTrackingIssues_KeepsTheLatest(t *testing.T) {
	now := time.Now().UTC()
	issues := []ErrorTrackingIssue{
		{ID: "older", LastSeen: now.Add(-2 * time.Hour)},
		{ID: "newest", LastSeen: now},
		{ID: "middle", LastSeen: now.Add(-time.Hour)},
		{ID: "unseen"},
	}

	newest := NewestErrorTrackingIssues(issues, 2)

	assert.Equal(t, []string{"newest", "middle"}, issueIDs(newest))
}

func TestNewestErrorTrackingIssues_KeepsEveryIssueWhenThePageIsSmall(t *testing.T) {
	now := time.Now().UTC()
	issues := []ErrorTrackingIssue{
		{ID: "older", LastSeen: now.Add(-time.Hour)},
		{ID: "newest", LastSeen: now},
	}

	newest := NewestErrorTrackingIssues(issues, 10)

	assert.Equal(t, []string{"newest", "older"}, issueIDs(newest))
}

func issueIDs(issues []ErrorTrackingIssue) []string {
	ids := make([]string, 0, len(issues))
	for _, issue := range issues {
		ids = append(ids, issue.ID)
	}
	return ids
}
