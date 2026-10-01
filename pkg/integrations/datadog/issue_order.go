package datadog

import (
	"slices"
	"time"
)

// NewestErrorTrackingIssues returns up to limit issues, most recently seen
// first. Issues with no last-seen time stay after issues that have one.
// Equal times keep their incoming order.
func NewestErrorTrackingIssues(issues []ErrorTrackingIssue, limit int) []ErrorTrackingIssue {
	return newestIssues(issues, limit, func(issue ErrorTrackingIssue) time.Time {
		return issue.LastSeen
	})
}

// NewestCreatedErrorTrackingIssues returns up to limit issues, most recently
// created first. Issues with no first-seen time stay after issues that have
// one. Equal times keep their incoming order.
func NewestCreatedErrorTrackingIssues(issues []ErrorTrackingIssue, limit int) []ErrorTrackingIssue {
	return newestIssues(issues, limit, func(issue ErrorTrackingIssue) time.Time {
		return issue.FirstSeen
	})
}

func newestIssues(issues []ErrorTrackingIssue, limit int, at func(ErrorTrackingIssue) time.Time) []ErrorTrackingIssue {
	if limit <= 0 || len(issues) == 0 {
		return nil
	}

	sorted := append([]ErrorTrackingIssue(nil), issues...)
	slices.SortStableFunc(sorted, func(left, right ErrorTrackingIssue) int {
		leftAt := at(left)
		rightAt := at(right)
		switch {
		case leftAt.Equal(rightAt):
			return 0
		case leftAt.After(rightAt):
			return -1
		default:
			return 1
		}
	})
	if len(sorted) > limit {
		return sorted[:limit]
	}
	return sorted
}
