package datadog

import "slices"

// NewestErrorTrackingIssues returns up to limit issues, most recently seen
// first. Issues with no last-seen time stay after issues that have one.
// Equal times keep their incoming order.
func NewestErrorTrackingIssues(issues []ErrorTrackingIssue, limit int) []ErrorTrackingIssue {
	if limit <= 0 || len(issues) == 0 {
		return nil
	}

	sorted := append([]ErrorTrackingIssue(nil), issues...)
	slices.SortStableFunc(sorted, func(left, right ErrorTrackingIssue) int {
		switch {
		case left.LastSeen.Equal(right.LastSeen):
			return 0
		case left.LastSeen.After(right.LastSeen):
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
