package bitbucket

import (
	"sort"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

const statusCheckRecentPRLimit = 3

func (b *Bitbucket) listStatusCheckResources(ctx core.ListResourcesContext, client *Client, repository string) ([]core.IntegrationResource, error) {
	repository = strings.TrimSpace(repository)
	if repository == "" {
		return []core.IntegrationResource{}, nil
	}
	if err := requireRepositoryInWorkspace(ctx.Integration, repository); err != nil {
		return nil, err
	}

	shas, err := recentStatusCheckSHAs(client, repository)
	if err != nil {
		return nil, err
	}
	var all []CommitStatus
	seenSHA := map[string]bool{}
	for _, sha := range shas {
		lower := strings.ToLower(strings.TrimSpace(sha))
		if lower == "" || seenSHA[lower] {
			continue
		}
		seenSHA[lower] = true
		statuses, err := client.ListCommitStatuses(repository, sha)
		if err != nil {
			return nil, err
		}
		all = append(all, statuses...)
	}

	if len(all) == 0 {
		branch, err := client.GetMainBranch(repository)
		if err != nil {
			return nil, err
		}
		branch = strings.TrimSpace(branch)
		if branch == "" {
			return []core.IntegrationResource{}, nil
		}
		head, err := client.GetBranchHead(repository, branch)
		if err != nil {
			return nil, err
		}
		head = strings.TrimSpace(head)
		if head == "" {
			return []core.IntegrationResource{}, nil
		}
		statuses, err := client.ListCommitStatuses(repository, head)
		if err != nil {
			return nil, err
		}
		all = statuses
	}

	return mergeStatusCheckResources(all), nil
}

func recentStatusCheckSHAs(client *Client, repository string) ([]string, error) {
	prs, err := client.ListPullRequests(repository, PullRequestListOptions{
		States:  []string{PullRequestStateOpen, PullRequestStateMerged, PullRequestStateDeclined},
		Sort:    "-updated_on",
		Pagelen: statusCheckRecentPRLimit,
	})
	if err != nil {
		return nil, err
	}
	// ponytail: recent-PRs-only discovery, full history if check catalog gaps matter
	shas := make([]string, 0, statusCheckRecentPRLimit)
	seen := map[string]bool{}
	for _, pr := range prs {
		sha := pullRequestSourceSHA(pr)
		trimmed := strings.TrimSpace(sha)
		if trimmed == "" {
			continue
		}
		lower := strings.ToLower(trimmed)
		if seen[lower] {
			continue
		}
		seen[lower] = true
		shas = append(shas, trimmed)
		if len(shas) >= statusCheckRecentPRLimit {
			break
		}
	}
	return shas, nil
}

func pullRequestSourceSHA(pr map[string]any) string {
	source, _ := pr["source"].(map[string]any)
	if source == nil {
		return ""
	}
	commit, _ := source["commit"].(map[string]any)
	if commit == nil {
		return ""
	}
	sha, _ := commit["hash"].(string)
	return strings.TrimSpace(sha)
}

func mergeStatusCheckResources(statuses []CommitStatus) []core.IntegrationResource {
	latest := map[string]CommitStatus{}
	latestUpdated := map[string]string{}
	for _, status := range statuses {
		key := strings.TrimSpace(status.Key)
		if key == "" {
			continue
		}
		lower := strings.ToLower(key)
		existing, ok := latest[lower]
		if !ok {
			latest[lower] = status
			latestUpdated[lower] = newestStatusTimestamp(status)
			continue
		}
		if isNewerStatus(status, existing) {
			latest[lower] = status
			latestUpdated[lower] = newestStatusTimestamp(status)
		} else if newestStatusTimestamp(status) == latestUpdated[lower] {
			// Keep the newest observed metadata; on ties prefer the later observation
			// only when it carries more complete display info.
			if strings.TrimSpace(status.Name) != "" && strings.TrimSpace(existing.Name) == "" {
				latest[lower] = status
			} else if strings.TrimSpace(status.URL) != "" && strings.TrimSpace(existing.URL) == "" {
				latest[lower] = status
			}
		}
	}

	resources := make([]core.IntegrationResource, 0, len(latest))
	for _, status := range latest {
		key := strings.TrimSpace(status.Key)
		if key == "" {
			continue
		}
		name := strings.TrimSpace(status.Name)
		if name == "" {
			name = key
		}
		resources = append(resources, core.IntegrationResource{
			Type: "status_check",
			ID:   key,
			Name: name,
			URL:  strings.TrimSpace(status.URL),
		})
	}
	sort.Slice(resources, func(i, j int) bool {
		lowerI := strings.ToLower(resources[i].ID)
		lowerJ := strings.ToLower(resources[j].ID)
		if lowerI != lowerJ {
			return lowerI < lowerJ
		}
		return resources[i].ID < resources[j].ID
	})
	return resources
}

func newestStatusTimestamp(status CommitStatus) string {
	updated := strings.TrimSpace(status.UpdatedOn)
	if updated != "" {
		return updated
	}
	return strings.TrimSpace(status.CreatedOn)
}

func isNewerStatus(next, current CommitStatus) bool {
	nextTS := newestStatusTimestamp(next)
	currentTS := newestStatusTimestamp(current)
	if nextTS == "" || currentTS == "" {
		return false
	}
	nextParsed, err := parseBitbucketTime(nextTS)
	if err != nil {
		return false
	}
	currentParsed, err := parseBitbucketTime(currentTS)
	if err != nil {
		return false
	}
	return nextParsed.After(currentParsed)
}
