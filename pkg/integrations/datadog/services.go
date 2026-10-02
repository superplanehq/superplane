package datadog

import (
	"errors"
	"slices"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

const (
	serviceFacet        = "service"
	serviceAggregateMax = 1000
)

// ListServices returns service names seen in spans or logs in the last 30
// days, plus services on open Error Tracking issues. A refused or failed
// telemetry query is skipped. The list fails only when every source fails.
// A refused issue search then returns ErrErrorTrackingForbidden.
func (c *Client) ListServices() ([]string, error) {
	spanNames, spanErr := c.aggregateFacet(spansAggregatePath, serviceFacet, "*", serviceAggregateMax)
	logNames, logErr := c.aggregateFacet(logsAggregatePath, serviceFacet, "*", serviceAggregateMax)
	issues, issueErr := c.SearchErrorTrackingIssues("*", maxErrorTrackingSearchLimit)
	if spanErr != nil && logErr != nil && issueErr != nil {
		return nil, serviceListError(issueErr, spanErr, logErr)
	}

	names := make([]string, 0)
	if spanErr == nil {
		names = append(names, spanNames...)
	}
	if logErr == nil {
		names = append(names, logNames...)
	}
	if issueErr == nil {
		names = append(names, serviceNamesFromIssues(issues)...)
	}
	return uniqueSortedNames(names), nil
}

func serviceListError(issueErr, spanErr, logErr error) error {
	if errors.Is(issueErr, ErrErrorTrackingForbidden) {
		return ErrErrorTrackingForbidden
	}
	if issueErr != nil {
		return issueErr
	}
	if spanErr != nil {
		return spanErr
	}
	return logErr
}

func serviceNamesFromIssues(issues []ErrorTrackingIssue) []string {
	names := make([]string, 0, len(issues))
	for _, issue := range issues {
		names = append(names, issue.Service)
	}
	return names
}

func uniqueSortedNames(names []string) []string {
	seen := map[string]struct{}{}
	unique := make([]string, 0, len(names))
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		unique = append(unique, name)
	}
	slices.Sort(unique)
	return unique
}

func serviceResources(names []string) []core.IntegrationResource {
	resources := make([]core.IntegrationResource, 0, len(names))
	for _, name := range names {
		resources = append(resources, core.IntegrationResource{
			Type: ResourceTypeService,
			ID:   name,
			Name: name,
		})
	}
	return resources
}
