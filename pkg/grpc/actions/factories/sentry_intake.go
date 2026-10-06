package factories

import (
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/superplanehq/superplane/pkg/integrations/sentry"
)

func normalizeSentryProjectIDs(ids []string) []string {
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

func sentryProjectIDsFromResource(resourceID string) []string {
	return normalizeSentryProjectIDs(strings.Split(resourceID, ","))
}

func sentryProjectIDsFromConfiguration(configuration map[string]any) []string {
	if configuration == nil {
		return nil
	}
	projects := normalizeSentryProjectIDs(configurationStrings(configuration["projects"]))
	if len(projects) > 0 {
		return projects
	}
	project, _ := configuration["project"].(string)
	return normalizeSentryProjectIDs([]string{project})
}

// applySentryProjectConfiguration writes the project list the intake listens
// to. A single project also sets project so older readers keep working. More
// than one project clears project, because TriggerResourceID prefers that
// string over the list.
func applySentryProjectConfiguration(configuration map[string]any, projectIDs []string) {
	configuration["projects"] = configurationAnyStrings(projectIDs)
	if len(projectIDs) == 1 {
		configuration["project"] = projectIDs[0]
		return
	}
	configuration["project"] = ""
}

func newestSentrySeedIssues(client *sentry.Client, projects []string) ([]sentry.Issue, error) {
	if len(projects) == 0 {
		return nil, fmt.Errorf("sentry intake has no projects")
	}

	matched := make([]sentry.Issue, 0, intakeSentrySeedSize)
	for _, project := range projects {
		issues, err := client.ListNewestUnresolvedIssues(project, intakeSentrySeedSize)
		if err != nil {
			return nil, fmt.Errorf("failed to list the issues of project %s: %w", project, err)
		}
		matched = append(matched, issues...)
	}
	return mergeNewestSentryIssues(matched, intakeSentrySeedSize), nil
}

func mergeNewestSentryIssues(issues []sentry.Issue, limit int) []sentry.Issue {
	if len(issues) == 0 {
		return issues
	}

	seen := map[string]bool{}
	merged := make([]sentry.Issue, 0, len(issues))
	for _, issue := range issues {
		if issue.ID != "" && seen[issue.ID] {
			continue
		}
		if issue.ID != "" {
			seen[issue.ID] = true
		}
		merged = append(merged, issue)
	}

	sort.SliceStable(merged, func(i, j int) bool {
		return sentryIssueSeenAt(merged[i]).After(sentryIssueSeenAt(merged[j]))
	})
	if limit > 0 && len(merged) > limit {
		return merged[:limit]
	}
	return merged
}

func sentryIssueSeenAt(issue sentry.Issue) time.Time {
	for _, value := range []string{issue.LastSeen, issue.FirstSeen} {
		parsed, err := time.Parse(time.RFC3339, value)
		if err == nil {
			return parsed
		}
	}
	return time.Time{}
}
