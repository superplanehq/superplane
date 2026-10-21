package runcodeagent

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
	githubcommon "github.com/superplanehq/superplane/pkg/integrations/github/common"
)

// IssueRef identifies a GitHub issue that a pull request should reference so
// that GitHub auto-closes it on merge.
type IssueRef struct {
	// Repository is the "owner/repo" the issue lives in.
	Repository string
	Number     int
}

// parseGitHubIssueOrigin reads owner/repo and issue number from a GitHub
// issue URL. Query strings and fragments are ignored.
func parseGitHubIssueOrigin(rawURL string) (ref IssueRef, ok bool) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || !githubcommon.IsKnownGitHubWebHostname(parsed.Hostname()) {
		return IssueRef{}, false
	}

	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(parts) < 4 || parts[0] == "" || parts[1] == "" || parts[2] != "issues" {
		return IssueRef{}, false
	}

	number, err := strconv.Atoi(parts[3])
	if err != nil {
		return IssueRef{}, false
	}

	return IssueRef{Repository: parts[0] + "/" + parts[1], Number: number}, true
}

// resolveResolvesIssue reports the GitHub issue the current work order was
// imported from, so the agent can be asked to add a "This resolves #N"
// backlink to the pull request it opens. It returns nil whenever the run has
// no work order, the work order has no origin, or the origin is not a GitHub
// issue (a GitHub pull request origin, a non-GitHub origin such as Linear, or
// a malformed URL all resolve to nil).
func resolveResolvesIssue(ctx core.ExecutionContext) *IssueRef {
	if ctx.Expressions == nil {
		return nil
	}

	value, err := ctx.Expressions.Run("order().origin")
	if err != nil || value == nil {
		return nil
	}

	origin, ok := value.(map[string]any)
	if !ok {
		return nil
	}

	url, _ := origin["url"].(string)
	ref, ok := parseGitHubIssueOrigin(url)
	if !ok {
		return nil
	}

	return &ref
}

// repositorySlug normalizes a runCodeAgent "repository" configuration value
// (owner/repo, or an https clone URL) down to "owner/repo".
// Returns "" when the value cannot be normalized (e.g. it is still an
// unresolved template expression).
func repositorySlug(repository string) string {
	repository = strings.TrimSuffix(strings.TrimSpace(repository), ".git")
	if repoOwnerRepoPattern.MatchString(repository) {
		return repository
	}
	host := githubHostFromRepository(repository)
	prefix := host.WebURL() + "/"
	if strings.HasPrefix(repository, prefix) {
		return strings.Trim(strings.TrimPrefix(repository, prefix), "/")
	}
	return ""
}

// issueBacklinkReference formats the origin issue reference for the
// "This resolves ..." backlink instruction: "#N" when the issue lives in the
// same repository as the pull request being opened, otherwise
// "owner/repo#N" so GitHub can still resolve the cross-repository reference.
func issueBacklinkReference(issue IssueRef, targetRepository string) string {
	if slug := repositorySlug(targetRepository); slug != "" && strings.EqualFold(slug, issue.Repository) {
		return fmt.Sprintf("#%d", issue.Number)
	}
	return fmt.Sprintf("%s#%d", issue.Repository, issue.Number)
}
