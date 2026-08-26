package factories

import (
	"testing"

	"github.com/google/go-github/v84/github"
	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/integrations/pagerduty"
	"github.com/superplanehq/superplane/pkg/integrations/sentry"
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

func TestSentryIssueItem_PrefersPermalink(t *testing.T) {
	item := sentryIssueItem(sentry.Issue{
		ID:        "7670162495",
		ShortID:   "API-12",
		Title:     "HTTP 500 on refunds",
		Permalink: "https://superplane.sentry.io/issues/7670162495/",
		WebURL:    "https://sentry.io/issues/7670162495/",
	})

	assert.Equal(t, IntakeItem{
		ID:    "7670162495",
		Key:   "API-12",
		Title: "HTTP 500 on refunds",
		Body:  "https://superplane.sentry.io/issues/7670162495/",
		URL:   "https://superplane.sentry.io/issues/7670162495/",
	}, item)
}

func TestPagerDutyIncidentItem_UsesIncidentNumber(t *testing.T) {
	item := pagerDutyIncidentItem(pagerduty.Incident{
		ID:             "P123ABC",
		IncidentNumber: 88,
		Title:          "Payments API down",
		Description:    "Latency spike",
		HTMLURL:        "https://acme.pagerduty.com/incidents/P123ABC",
	})

	assert.Equal(t, IntakeItem{
		ID:    "P123ABC",
		Key:   "#88",
		Title: "Payments API down",
		Body:  "Latency spike",
		URL:   "https://acme.pagerduty.com/incidents/P123ABC",
	}, item)
}

func TestFilterIntakeItems_MatchesKeyTitleAndBody(t *testing.T) {
	items := []IntakeItem{
		{ID: "1", Key: "#12", Title: "Handle duplicate refunds", Body: "Retrying a refund posts twice."},
		{ID: "2", Key: "#13", Title: "Add logging", Body: "Trace the checkout path."},
	}

	assert.Equal(t, items[:1], filterIntakeItems(items, "refund", 5))
	assert.Equal(t, items[:1], filterIntakeItems(items, "#12", 5))
	assert.Equal(t, items, filterIntakeItems(items, "", 5))
	assert.Len(t, filterIntakeItems(items, "", 1), 1)
}

func TestIntakeItemLimit(t *testing.T) {
	assert.Equal(t, defaultLatestIntakeItems, intakeItemLimit("", 0))
	assert.Equal(t, defaultSearchIntakeItems, intakeItemLimit("refund", 0))
	assert.Equal(t, 3, intakeItemLimit("", 3))
	assert.Equal(t, maxIntakeItems, intakeItemLimit("refund", 100))
}

func TestPageIntakeItems_SkipsOffsetAndReportsMore(t *testing.T) {
	items := []IntakeItem{
		{ID: "1", Title: "One"},
		{ID: "2", Title: "Two"},
		{ID: "3", Title: "Three"},
	}

	page, hasMore := pageIntakeItems(items, "", 2, 0)
	assert.Equal(t, items[:2], page)
	assert.True(t, hasMore)

	page, hasMore = pageIntakeItems(items, "", 2, 2)
	assert.Equal(t, items[2:], page)
	assert.False(t, hasMore)
}

func TestGitHubIssueSearchQuery_QuotesTheOperatorTerm(t *testing.T) {
	assert.Equal(t, `repo:acme/pay is:issue is:open "refund"`, gitHubIssueSearchQuery("acme/pay", "refund"))
	assert.Equal(
		t,
		`repo:acme/pay is:issue is:open "repo:other/repo org:evil"`,
		gitHubIssueSearchQuery("acme/pay", "repo:other/repo org:evil"),
	)
}

func TestSentryIssueInProject_RequiresConfiguredSlug(t *testing.T) {
	source := &liveIntakeItemSource{sentryProject: "payments-api"}
	assert.False(t, source.sentryIssueInProject(&sentry.Issue{ID: "1"}))
	assert.False(t, source.sentryIssueInProject(&sentry.Issue{
		ID:      "1",
		Project: &sentry.IssueProject{Slug: "other-app"},
	}))
	assert.True(t, source.sentryIssueInProject(&sentry.Issue{
		ID:      "1",
		Project: &sentry.IssueProject{Slug: "payments-api"},
	}))

	unscoped := &liveIntakeItemSource{}
	assert.True(t, unscoped.sentryIssueInProject(&sentry.Issue{ID: "1"}))
}
