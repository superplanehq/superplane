package factories

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/go-github/v84/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/integrations/datadog"
	"github.com/superplanehq/superplane/pkg/integrations/jira"
	"github.com/superplanehq/superplane/pkg/integrations/sentry"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestGitHubIssueItem_UsesNumberKeyAndHTMLURL(t *testing.T) {
	number := 12
	title := "Handle duplicate refunds"
	body := "Retrying a refund posts twice."
	url := "https://github.com/acme/payments/issues/12"
	issue := &github.Issue{
		Number:  &number,
		Title:   &title,
		Body:    &body,
		HTMLURL: &url,
	}

	assert.Equal(t, IntakeItem{
		ID:    "12",
		Key:   "#12",
		Title: title,
		Body:  body,
		URL:   url,
	}, gitHubIssueItem(issue))
}

func dependabotTestAlert(number int, name, manifest, severity, state string) *github.DependabotAlert {
	return &github.DependabotAlert{
		Number:  github.Ptr(number),
		State:   github.Ptr(state),
		HTMLURL: github.Ptr("https://github.com/acme/payments/security/dependabot/" + strconv.Itoa(number)),
		Dependency: &github.Dependency{
			Package:      &github.VulnerabilityPackage{Name: github.Ptr(name), Ecosystem: github.Ptr("npm")},
			ManifestPath: github.Ptr(manifest),
		},
		SecurityAdvisory: &github.DependabotSecurityAdvisory{
			Summary:  github.Ptr("Vulnerability in " + name),
			Severity: github.Ptr(severity),
		},
	}
}

func TestDependabotPackageGroups_OneGroupPerPackage(t *testing.T) {
	groups := dependabotPackageGroups("acme/payments", []*github.DependabotAlert{
		dependabotTestAlert(12, "astro", "package-lock.json", "high", "open"),
		dependabotTestAlert(11, "fflate", "package-lock.json", "medium", "open"),
		dependabotTestAlert(10, "astro", "package.json", "high", "open"),
		dependabotTestAlert(9, "astro", "package.json", "low", "fixed"),
	}, nil)

	require.Len(t, groups, 2)
	assert.Equal(t, "astro", groups[0].ref.Name)
	assert.Len(t, groups[0].alerts, 2)
	assert.Equal(t, "fflate", groups[1].ref.Name)
	assert.Len(t, groups[1].alerts, 1)

	item := dependabotPackageItem(groups[0])
	assert.Equal(t, "npm:astro", item.ID)
	assert.Equal(t, "2 alerts", item.Key)
	assert.Equal(t, "Fix Dependabot alerts for astro (npm)", item.Title)
	assert.Contains(t, item.Body, "### #12 Vulnerability in astro")
	assert.Contains(t, item.Body, "### #10 Vulnerability in astro")
	assert.Equal(t, "https://github.com/acme/payments/security/dependabot?q=is%3Aopen+package%3Aastro+ecosystem%3Anpm", item.URL)
	assert.Equal(t, "1 alert", dependabotPackageItem(groups[1]).Key)
}

func TestDependabotPackageGroups_KeepsConfiguredSeverities(t *testing.T) {
	groups := dependabotPackageGroups("acme/payments", []*github.DependabotAlert{
		dependabotTestAlert(12, "astro", "package-lock.json", "high", "open"),
		dependabotTestAlert(11, "fflate", "package-lock.json", "medium", "open"),
		dependabotTestAlert(10, "astro", "package.json", "low", "open"),
	}, []string{"critical", "high"})

	require.Len(t, groups, 1)
	assert.Equal(t, "astro", groups[0].ref.Name)
	assert.Len(t, groups[0].alerts, 1)
}

func TestDependabotPackageRefFromItemID(t *testing.T) {
	ref, ok := dependabotPackageRefFromItemID("acme/payments", "npm:@babel/core")
	require.True(t, ok)
	assert.Equal(t, "npm", ref.Ecosystem)
	assert.Equal(t, "@babel/core", ref.Name)
	assert.Equal(t, "acme/payments", ref.Repository)

	_, ok = dependabotPackageRefFromItemID("acme/payments", "7")
	assert.False(t, ok)
}

func TestIntakeItemLimit(t *testing.T) {
	assert.Equal(t, defaultLatestIntakeItems, intakeItemLimit("", 0))
	assert.Equal(t, defaultSearchIntakeItems, intakeItemLimit("refund", 0))
	assert.Equal(t, 3, intakeItemLimit("", 3))
	assert.Equal(t, maxIntakeItems, intakeItemLimit("refund", 100))
}

func TestGitHubIssueSearchQuery_QuotesTheOperatorTerm(t *testing.T) {
	assert.Equal(t, "repo:acme/pay is:issue is:open", gitHubIssueSearchQuery("acme/pay", ""))
	assert.Equal(t, `repo:acme/pay is:issue is:open "refund"`, gitHubIssueSearchQuery("acme/pay", "refund"))
	assert.Equal(
		t,
		`repo:acme/pay is:issue is:open "repo:other/repo org:evil"`,
		gitHubIssueSearchQuery("acme/pay", "repo:other/repo org:evil"),
	)
}

func TestComposeImportedDescription(t *testing.T) {
	now := time.Now().UTC()
	earlier := now.Add(-2 * time.Hour)
	later := now.Add(-1 * time.Hour)

	issueComment := func(login, body string, createdAt time.Time) *github.IssueComment {
		comment := &github.IssueComment{
			Body:      github.Ptr(body),
			CreatedAt: &github.Timestamp{Time: createdAt},
		}
		if login != "" {
			comment.User = &github.User{Login: github.Ptr(login)}
		}
		return comment
	}

	t.Run("body and comments oldest first, separated by a blank line", func(t *testing.T) {
		body := "A retried refund charges the customer twice."
		comments := []*github.IssueComment{
			issueComment("ana", "Confirmed on staging.", earlier),
			issueComment("bruno", "Let's cap retries at 3.", later),
		}

		expected := fmt.Sprintf(
			"%s\n%s\n%s\nana %s\nConfirmed on staging.\n\nbruno %s\nLet's cap retries at 3.",
			body,
			importedCommentsSeparator,
			importedCommentsHeader,
			earlier.Format(time.RFC3339),
			later.Format(time.RFC3339),
		)

		assert.Equal(t, expected, composeImportedDescription(body, comments))
	})

	t.Run("no comments returns the body unchanged", func(t *testing.T) {
		body := "A retried refund charges the customer twice."

		result := composeImportedDescription(body, nil)

		assert.Equal(t, body, result)
		assert.NotContains(t, result, importedCommentsSeparator)
		assert.NotContains(t, result, importedCommentsHeader)
	})

	t.Run("empty body starts directly with the separator", func(t *testing.T) {
		comments := []*github.IssueComment{issueComment("ana", "Confirmed on staging.", earlier)}

		result := composeImportedDescription("", comments)

		expected := fmt.Sprintf(
			"%s\n%s\nana %s\nConfirmed on staging.",
			importedCommentsSeparator,
			importedCommentsHeader,
			earlier.Format(time.RFC3339),
		)
		assert.Equal(t, expected, result)
		assert.False(t, strings.HasPrefix(result, "\n"))
	})

	t.Run("comment with no user falls back to a placeholder author", func(t *testing.T) {
		comments := []*github.IssueComment{issueComment("", "Anonymous comment.", earlier)}

		result := composeImportedDescription("Body text", comments)

		assert.Contains(t, result, fmt.Sprintf("%s %s\nAnonymous comment.", importedCommentsUnknownAuthor, earlier.Format(time.RFC3339)))
	})

	t.Run("nil comments are ignored", func(t *testing.T) {
		body := "Body text"
		comments := []*github.IssueComment{nil}

		assert.Equal(t, body, composeImportedDescription(body, comments))
	})
}

func TestJiraIntakeItemSource_StaysInsideItsProject(t *testing.T) {
	source := &jiraIntakeItemSource{projectKey: "ENG", siteURL: "https://acme.atlassian.net"}

	t.Run("an origin URL of another project of the same site is not this intake's", func(t *testing.T) {
		id, ok := source.ItemIDFromOriginURL("https://acme.atlassian.net/browse/ENG-42")
		assert.True(t, ok)
		assert.Equal(t, "ENG-42", id)

		_, ok = source.ItemIDFromOriginURL("https://acme.atlassian.net/browse/OPS-42")
		assert.False(t, ok)

		_, ok = source.ItemIDFromOriginURL("https://other.atlassian.net/browse/ENG-42")
		assert.False(t, ok)
	})

	t.Run("an issue reports as owned only with a matching project key", func(t *testing.T) {
		assert.True(t, source.ownsIssue(map[string]any{"project": map[string]any{"key": "eng"}}))
		assert.False(t, source.ownsIssue(map[string]any{"project": map[string]any{"key": "OPS"}}))
		assert.False(t, source.ownsIssue(map[string]any{"project": map[string]any{}}))
		assert.False(t, source.ownsIssue(nil))
	})
}

func TestJiraIssueProjectKey(t *testing.T) {
	assert.Equal(t, "ENG", jiraIssueProjectKey("ENG-42"))
	assert.Equal(t, "ENG-SUB", jiraIssueProjectKey("ENG-SUB-42"))
	assert.Equal(t, "", jiraIssueProjectKey("ENG42"))
	assert.Equal(t, "", jiraIssueProjectKey("-42"))
}

func TestJiraIssueFromFullIssue_ReadsTheDescriptionAsText(t *testing.T) {
	issue := &jira.Issue{
		Key: "ENG-42",
		Fields: map[string]any{
			"summary": "Refund retries charge twice",
			"description": map[string]any{
				"type":    "doc",
				"version": float64(1),
				"content": []any{
					map[string]any{
						"type":    "paragraph",
						"content": []any{map[string]any{"type": "text", "text": "A retried refund charges twice."}},
					},
				},
			},
		},
	}

	assert.Equal(t, IntakeItem{
		ID:    "ENG-42",
		Key:   "ENG-42",
		Title: "Refund retries charge twice",
		Body:  "A retried refund charges twice.",
		URL:   "https://acme.atlassian.net/browse/ENG-42",
	}, jiraIssueFromFullIssue(issue, "https://acme.atlassian.net"))
}

func TestUnsupportedIntakeItemSource_DoesNotSearch(t *testing.T) {
	source := unsupportedIntakeItemSource{}
	_, err := source.Search(t.Context(), "refund", 5)
	assert.ErrorIs(t, err, errIntakeSearchUnsupported)
	_, err = source.Get(t.Context(), "1")
	assert.ErrorIs(t, err, errIntakeSearchUnsupported)
}

func TestSentryIssueItem_UsesShortIDAndPermalink(t *testing.T) {
	issue := sentry.Issue{
		ID:        "123",
		ShortID:   "PAYMENTS-1",
		Title:     "TypeError: boom",
		Permalink: "https://acme.sentry.io/issues/123/",
		WebURL:    "https://sentry.io/issues/123/",
	}

	assert.Equal(t, IntakeItem{
		ID:    "123",
		Key:   "PAYMENTS-1",
		Title: "TypeError: boom",
		URL:   "https://acme.sentry.io/issues/123/",
	}, sentryIssueItem(issue))
}

func TestSentryIssueItem_FallsBackToWebURL(t *testing.T) {
	issue := sentry.Issue{
		ID:      "123",
		ShortID: "PAYMENTS-1",
		Title:   "TypeError: boom",
		WebURL:  "https://sentry.io/issues/123/",
	}

	assert.Equal(t, IntakeItem{
		ID:    "123",
		Key:   "PAYMENTS-1",
		Title: "TypeError: boom",
		URL:   "https://sentry.io/issues/123/",
	}, sentryIssueItem(issue))
}

func TestSentryIntakeItemSource_StaysInsideItsProject(t *testing.T) {
	source := &sentryIntakeItemSource{project: "payments"}

	assert.True(t, source.ownsIssue(&sentry.Issue{Project: &sentry.IssueProject{Slug: "payments"}}))
	assert.True(t, source.ownsIssue(&sentry.Issue{Project: &sentry.IssueProject{Slug: "PAYMENTS"}}))
	assert.False(t, source.ownsIssue(&sentry.Issue{Project: &sentry.IssueProject{Slug: "billing"}}))
	assert.False(t, source.ownsIssue(&sentry.Issue{Project: &sentry.IssueProject{}}))
	assert.False(t, source.ownsIssue(&sentry.Issue{}))
	assert.False(t, source.ownsIssue(nil))
}

func TestDatadogIntakeItemSource_StaysInsideItsService(t *testing.T) {
	source := &datadogIntakeItemSource{service: "checkout"}

	assert.True(t, source.ownsIssue(datadog.ErrorTrackingIssue{Service: "checkout"}))
	assert.True(t, source.ownsIssue(datadog.ErrorTrackingIssue{Service: "Checkout"}))
	assert.False(t, source.ownsIssue(datadog.ErrorTrackingIssue{Service: "billing"}))
	assert.True(t, source.ownsIssue(datadog.ErrorTrackingIssue{Service: ""}))
	assert.True(t, source.ownsIssue(datadog.ErrorTrackingIssue{}))
}

func TestDatadogServiceSearchQuery(t *testing.T) {
	assert.Equal(t, "service:checkout", datadogServiceSearchQuery("checkout", ""))
	assert.Equal(t, "service:checkout", datadogServiceSearchQuery(" checkout ", "  "))
	assert.Equal(t, "service:checkout timeout", datadogServiceSearchQuery("checkout", "timeout"))
}

func TestDatadogIntakeItemSource_GetIncludesSampleAndRelatedLogs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/api/v2/error-tracking/issues/"):
			_, _ = w.Write([]byte(`{"data":{"id":"issue-1","type":"issue","attributes":{"error_type":"TimeoutError","error_message":"checkout timed out","service":"checkout","file_path":"checkout/pay.go","function_name":"Charge","platform":"BACKEND"}}}`))
		case r.URL.Path == "/api/v2/error-tracking/issues/search":
			_, _ = w.Write([]byte(`{"data":[]}`))
		case r.URL.Path == "/api/v2/spans/events/search":
			_, _ = w.Write([]byte(`{"data":[{"attributes":{"custom":{"otel":{"trace_id":"cf2c57cfc127be5a1f156480875acc0a"},"env":"development","resource_name":"POST /checkout","error":{"stack":"goroutine 1 [running]:\nmain.Charge(checkout/pay.go:22)"}},"start_timestamp":"2026-09-27T19:51:32.010Z"}}]}`))
		case r.URL.Path == "/api/v2/logs/events/search":
			_, _ = w.Write([]byte(`{"data":[{"attributes":{"timestamp":"2026-09-27T19:51:32.010Z","status":"error","service":"checkout","message":"POST /checkout -> 500 (0.2ms)"}}]}`))
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client, err := datadog.NewClient(server.Client(), &contexts.IntegrationContext{
		Configuration: map[string]any{"site": "datadoghq.eu", "apiKey": "api-key", "appKey": "app-key"},
	})
	require.NoError(t, err)
	client.BaseURL = server.URL

	source := &datadogIntakeItemSource{datadog: client, service: "checkout"}
	item, err := source.Get(t.Context(), "issue-1")
	require.NoError(t, err)
	require.NotNil(t, item)
	assert.Equal(t, "TimeoutError: checkout timed out", item.Title)
	assert.Contains(t, item.Body, "## Error sample")
	assert.Contains(t, item.Body, "POST /checkout")
	assert.Contains(t, item.Body, "## Related logs")
	assert.Contains(t, item.Body, "POST /checkout -> 500 (0.2ms)")
}

func TestDatadogIssueItem_UsesServiceAndAppURL(t *testing.T) {
	client := &datadog.Client{Site: "datadoghq.eu"}
	issue := datadog.ErrorTrackingIssue{
		ID:           "issue-1",
		ErrorType:    "TimeoutError",
		ErrorMessage: "checkout timed out",
		Service:      "checkout",
	}

	assert.Equal(t, IntakeItem{
		ID:    "issue-1",
		Key:   "checkout",
		Title: "TimeoutError: checkout timed out",
		Body:  "checkout timed out",
		URL:   "https://app.datadoghq.eu/error-tracking/issue/issue-1",
	}, datadogIssueItem(client, issue))
}
