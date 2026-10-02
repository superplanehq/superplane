package datadog

import (
	"errors"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/superplanehq/superplane/pkg/core"
)

const (
	serviceFacet             = "service"
	serviceAggregatePageSize = 1000
	// serviceAggregateMax is the Datadog logs group_by maximum. A full page
	// at the smaller size asks for this many names so a long list is not cut
	// at the first thousand.
	serviceAggregateMax = 10000
	// serviceListBudget is one outbound request timeout. The three reads share
	// it so a stalled span or log query cannot add another full timeout
	// before the issue search runs.
	serviceListBudget        = 30 * time.Second
	serviceAggregateMaxPages = 10
)

// ListServices returns service names seen in spans or logs in the last 30
// days, plus services on open Error Tracking issues. The three reads run
// together under one time limit. A refused or failed telemetry query is
// skipped. The list fails only when every source fails. A refused issue
// search then returns ErrErrorTrackingForbidden.
func (c *Client) ListServices() ([]string, error) {
	restoreDeadline := c.applyServiceListDeadline()
	defer restoreDeadline()

	var (
		spanNames []string
		logNames  []string
		issues    []ErrorTrackingIssue
		spanErr   error
		logErr    error
		issueErr  error
	)
	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer wg.Done()
		spanNames, spanErr = c.listTelemetryServices(spansAggregatePath)
	}()
	go func() {
		defer wg.Done()
		logNames, logErr = c.listTelemetryServices(logsAggregatePath)
	}()
	go func() {
		defer wg.Done()
		issues, issueErr = c.SearchErrorTrackingIssues("*", maxErrorTrackingSearchLimit)
	}()
	wg.Wait()

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

func (c *Client) applyServiceListDeadline() func() {
	if !c.requestDeadline.IsZero() {
		return func() {}
	}
	c.requestDeadline = time.Now().Add(serviceListBudget)
	return func() {
		c.requestDeadline = time.Time{}
	}
}

func (c *Client) serviceListDeadlinePassed() bool {
	return !c.requestDeadline.IsZero() && !time.Now().Before(c.requestDeadline)
}

// listTelemetryServices reads one telemetry source. A short first page is the
// full list. A full page asks for the API maximum, then continues by name
// while Datadog keeps returning new names.
func (c *Client) listTelemetryServices(path string) ([]string, error) {
	names, err := c.aggregateFacet(path, serviceFacet, "*", serviceAggregatePageSize)
	if err != nil || len(names) < serviceAggregatePageSize || c.serviceListDeadlinePassed() {
		return names, err
	}

	wider, widerErr := c.aggregateFacet(path, serviceFacet, "*", serviceAggregateMax)
	if widerErr != nil {
		return c.pageServiceNames(path, names, serviceAggregatePageSize)
	}
	names = append(names, wider...)
	if len(wider) < serviceAggregateMax || c.serviceListDeadlinePassed() {
		return names, nil
	}
	return c.pageServiceNames(path, names, serviceAggregateMax)
}

func (c *Client) pageServiceNames(path string, names []string, limit int) ([]string, error) {
	seen := map[string]struct{}{}
	collected := make([]string, 0, len(names))
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		collected = append(collected, name)
	}
	if len(collected) == 0 {
		return collected, nil
	}

	for page := 1; page < serviceAggregateMaxPages; page++ {
		if c.serviceListDeadlinePassed() {
			return collected, nil
		}
		last := slices.Max(collected)
		batch, err := c.aggregateFacet(path, serviceFacet, nextServiceQuery(last), limit)
		if err != nil {
			return collected, nil
		}
		added := 0
		for _, name := range batch {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			if _, ok := seen[name]; ok {
				continue
			}
			seen[name] = struct{}{}
			collected = append(collected, name)
			added++
		}
		if added == 0 || len(batch) < limit {
			return collected, nil
		}
	}
	return collected, nil
}

func nextServiceQuery(last string) string {
	return "service:>" + quoteQueryValue(last)
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
