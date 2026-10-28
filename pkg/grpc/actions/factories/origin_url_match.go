package factories

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/superplanehq/superplane/pkg/integrations/datadog"
	"github.com/superplanehq/superplane/pkg/integrations/jira"
	"github.com/superplanehq/superplane/pkg/integrations/linear"
	"github.com/superplanehq/superplane/pkg/integrations/sentry"
	"github.com/superplanehq/superplane/pkg/models"
)

var errAbsoluteOriginURL = errors.New("source_url must be an absolute http URL")

// workOrderOriginMatch builds the list predicate for one recorded issue URL.
// Known intake URLs match that issue identity. Other absolute URLs match the
// normalized recorded URL. A bare issue number or key is rejected.
func workOrderOriginMatch(raw string) (*models.WorkOrderOriginMatch, error) {
	raw = strings.TrimSpace(raw)
	parsed, err := absoluteHTTPURL(raw)
	if err != nil {
		return nil, err
	}

	if host, repository, number, ok := parseGitHubIssueURL(raw); ok {
		owner, repo, found := strings.Cut(repository, "/")
		if found && owner != "" && repo != "" {
			pattern := fmt.Sprintf(
				`^https?://%s/%s/%s/issues/%d/?([?#].*)?$`,
				posixQuote(host.WebHostname()), posixQuote(owner), posixQuote(repo), number,
			)
			return originRegexpMatch(pattern), nil
		}
	}

	if ref, ok := linear.IssueRefFromURL(raw); ok {
		pattern := fmt.Sprintf(
			`^https?://%s/[^/?#]+/issue/%s(/[^?#]*)?/?([?#].*)?$`,
			posixQuote(ref.Host), posixQuote(ref.Identifier),
		)
		return originRegexpMatch(pattern), nil
	}

	if ref, ok := jira.IssueRefFromURL(raw); ok {
		pattern := fmt.Sprintf(
			`^https?://%s/browse/%s/?([?#].*)?$`,
			posixQuote(ref.Host), posixQuote(ref.Key),
		)
		return originRegexpMatch(pattern), nil
	}

	if issueID, ok := sentry.IssueIDFromURL(raw); ok {
		pattern := fmt.Sprintf(
			`^https?://%s(/organizations/[^/?#]+)?/issues/%s(/[^?#]*)?/?([?#].*)?$`,
			posixQuote(parsed.Hostname()), posixQuote(issueID),
		)
		return originRegexpMatch(pattern), nil
	}

	if issueID, ok := datadog.IssueIDFromURL(raw); ok {
		pattern := fmt.Sprintf(
			`^https?://%s/[^?#]*error-tracking/issue/%s(/[^?#]*)?([?#].*)?$`,
			posixQuote(parsed.Hostname()), posixQuote(issueID),
		)
		return originRegexpMatch(pattern), nil
	}

	if organizationID, taskID, ok := parseProductiveTaskURL(raw); ok {
		pattern := fmt.Sprintf(
			`^https?://app\.productive\.io/%s/tasks/%s/?([?#].*)?$`,
			posixQuote(organizationID), posixQuote(taskID),
		)
		return originRegexpMatch(pattern), nil
	}

	normalized, ok := normalizeOriginURL(parsed)
	if !ok {
		return nil, errAbsoluteOriginURL
	}
	return &models.WorkOrderOriginMatch{
		SQL:  normalizedOriginSQL + " = ?",
		Args: []any{normalized},
	}, nil
}

func originRegexpMatch(pattern string) *models.WorkOrderOriginMatch {
	return &models.WorkOrderOriginMatch{
		SQL:  "factory_work_orders.origin_url ~* ?",
		Args: []any{pattern},
	}
}

func absoluteHTTPURL(raw string) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Hostname() == "" {
		return nil, errAbsoluteOriginURL
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, errAbsoluteOriginURL
	}
	return parsed, nil
}

// normalizedOriginSQL matches normalizeOriginURL.
// The host is lowercase, the fragment is dropped, and a trailing slash is dropped.
// chr() keeps the question mark out of the SQL text so GORM does not treat it as a placeholder.
const normalizedOriginSQL = `(` +
	`lower(substring(btrim(factory_work_orders.origin_url) from '^[^:]+'))` +
	` || '://' || ` +
	`lower(substring(btrim(factory_work_orders.origin_url) from '://([^/]+)'))` +
	` || regexp_replace(` +
	`split_part(coalesce(substring(btrim(factory_work_orders.origin_url) from '://[^/]+(.*)'), ''), '#', 1),` +
	` '/+($|' || chr(92) || chr(63) || ')',` +
	` chr(92) || '1'` +
	`)` +
	`)`

func normalizeOriginURL(parsed *url.URL) (string, bool) {
	if parsed == nil || parsed.Hostname() == "" {
		return "", false
	}
	path := strings.TrimRight(parsed.EscapedPath(), "/")
	normalized := strings.ToLower(parsed.Scheme) + "://" + strings.ToLower(parsed.Host) + path
	if parsed.RawQuery != "" {
		normalized += "?" + parsed.RawQuery
	}
	return normalized, true
}

func posixQuote(value string) string {
	var out strings.Builder
	for _, r := range value {
		switch r {
		case '\\', '.', '^', '$', '*', '+', '?', '(', ')', '[', ']', '{', '}', '|':
			out.WriteByte('\\')
		}
		out.WriteRune(r)
	}
	return out.String()
}
