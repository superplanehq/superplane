package bitbucket

import (
	"fmt"
	"maps"
	"regexp"
	"strings"
)

// ForgeEventKey maps the PR lifecycle and build-status events available to
// Forge to repository webhook keys.
func ForgeEventKey(eventType string) string {
	switch eventType {
	case "avi:bitbucket:created:pullrequest":
		return pullRequestEventCreated
	case "avi:bitbucket:updated:pullrequest":
		return pullRequestEventUpdated
	case "avi:bitbucket:fulfilled:pullrequest":
		return pullRequestEventFulfilled
	case "avi:bitbucket:rejected:pullrequest":
		return pullRequestEventRejected
	case "avi:bitbucket:created:build-status":
		return waitBuildsStatusCreated
	case "avi:bitbucket:updated:build-status":
		return waitBuildsStatusUpdated
	case "avi:bitbucket:created:pullrequest-comment":
		return pullRequestCommentCreated
	default:
		return ""
	}
}

// IsForgeBuildEvent reports whether a Forge event carries a build status.
func IsForgeBuildEvent(eventType string) bool {
	return eventType == "avi:bitbucket:created:build-status" || eventType == "avi:bitbucket:updated:build-status"
}

// IsForgePullRequestEvent reports whether a Forge event carries a pull request.
func IsForgePullRequestEvent(eventType string) bool {
	switch eventType {
	case "avi:bitbucket:created:pullrequest", "avi:bitbucket:updated:pullrequest",
		"avi:bitbucket:fulfilled:pullrequest", "avi:bitbucket:rejected:pullrequest":
		return true
	default:
		return false
	}
}

// ForgePullRequestPayload projects Forge's event-time fields into the repository webhook contract.
func ForgePullRequestPayload(eventType, timestamp, repositoryUUID, fullName string, pullRequest, actor map[string]any) map[string]any {
	pr := maps.Clone(pullRequest)
	for _, field := range []struct{ forge, webhook string }{
		{"createdOn", "created_on"}, {"updatedOn", "updated_on"}, {"mergeCommit", "merge_commit"},
	} {
		if value, ok := pr[field.forge]; ok {
			pr[field.webhook] = value
			delete(pr, field.forge)
		}
	}
	if updated, _ := pr["updated_on"].(string); updated == "" {
		pr["updated_on"] = timestamp
	}
	if title, ok := pr["title"].(map[string]any); ok {
		pr["title"] = title["value"]
	}
	for _, side := range []string{"source", "destination"} {
		if value, ok := pr[side].(map[string]any); ok {
			branch := maps.Clone(value)
			if name, ok := branch["branch"].(string); ok {
				branch["branch"] = map[string]any{"name": name}
			}
			pr[side] = branch
		}
	}
	if author, ok := pr["author"].(map[string]any); ok {
		author = maps.Clone(author)
		author["display_name"] = author["displayName"]
		pr["author"] = author
	}
	switch ForgeEventKey(eventType) {
	case pullRequestEventFulfilled:
		pr["state"] = "MERGED"
	case pullRequestEventRejected:
		pr["state"] = "DECLINED"
	}
	id, _ := int64FromJSON(pr["id"])
	pr["links"] = map[string]any{"html": map[string]any{"href": fmt.Sprintf("https://bitbucket.org/%s/pull-requests/%d", fullName, id)}}
	return map[string]any{
		"pullrequest": pr,
		"repository":  map[string]any{"uuid": repositoryUUID, "full_name": fullName},
		"actor":       actor,
	}
}

// ForgeBuildStatusPayload projects Forge's buildStatus object into the
// repository commit_status webhook contract the waiter already handles.
// Repository identity comes from the matched subscription metadata, not the
// Forge payload, so events route to the configured repository.
func ForgeBuildStatusPayload(timestamp, repositoryUUID, fullName string, buildStatus, actor map[string]any) map[string]any {
	status := maps.Clone(buildStatus)
	for _, field := range []struct{ forge, webhook string }{
		{"createdOn", "created_on"}, {"updatedOn", "updated_on"},
	} {
		if value, ok := status[field.forge]; ok {
			status[field.webhook] = value
			delete(status, field.forge)
		}
	}
	if updated, _ := status["updated_on"].(string); updated == "" && strings.TrimSpace(timestamp) != "" {
		status["updated_on"] = timestamp
	}
	if created, _ := status["created_on"].(string); created == "" && strings.TrimSpace(timestamp) != "" {
		status["created_on"] = timestamp
	}
	commitHash := ForgeBuildCommitSHA(status)
	commit := map[string]any{"hash": commitHash}
	// Keep the nested commit shape the waiter reads, plus a flat hash for debuggability.
	status["commit"] = commit
	return map[string]any{
		"repository":    map[string]any{"uuid": repositoryUUID, "full_name": fullName},
		"commit_status": status,
		"actor":         actor,
	}
}

// ForgeBuildCommitSHA extracts the full commit hash from a Forge buildStatus object.
func ForgeBuildCommitSHA(buildStatus map[string]any) string {
	if buildStatus == nil {
		return ""
	}
	commit, _ := buildStatus["commit"].(map[string]any)
	hash, _ := commit["hash"].(string)
	return strings.TrimSpace(hash)
}

// ForgeBuildKey extracts the build key from a Forge buildStatus object.
func ForgeBuildKey(buildStatus map[string]any) string {
	if buildStatus == nil {
		return ""
	}
	key, _ := buildStatus["key"].(string)
	return strings.TrimSpace(key)
}

var fullCommitSHARegex = regexp.MustCompile(`^[0-9a-fA-F]{40}$`)

// IsValidFullCommitSHA reports whether a value is a 40-character hex commit SHA.
func IsValidFullCommitSHA(sha string) bool {
	return fullCommitSHARegex.MatchString(strings.TrimSpace(sha))
}
