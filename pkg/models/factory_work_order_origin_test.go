package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOriginFromIntakePayload_ReadsNestedHTTPURL(t *testing.T) {
	origin := OriginFromIntakePayload(map[string]any{
		"issue": map[string]any{
			"html_url": "https://github.com/acme/payments/issues/12",
			"title":    "Handle duplicate refunds",
		},
	})

	assert.Equal(t, &WorkOrderOrigin{
		URL:   "https://github.com/acme/payments/issues/12",
		Label: "acme/payments#12",
	}, origin)
}

func TestOriginFromIntakePayload_PrefersPermalinkOverOtherURLs(t *testing.T) {
	origin := OriginFromIntakePayload(map[string]any{
		"data": map[string]any{
			"issue": map[string]any{
				"permalink": "https://example.com/issues/7670162495/",
				"web_url":   "https://example.com/other/7670162495/",
			},
		},
	})

	assert.Equal(t, &WorkOrderOrigin{
		URL:   "https://example.com/issues/7670162495/",
		Label: "7670162495",
	}, origin)
}

func TestOriginFromIntakeRootEvent_PeelsEnvelopeThenReadsURL(t *testing.T) {
	event := &CanvasEvent{Data: NewJSONValue(map[string]any{
		"type": "github.issue",
		"data": map[string]any{
			"issue": map[string]any{"html_url": "https://github.com/acme/payments/issues/12"},
		},
	})}

	assert.Equal(t, &WorkOrderOrigin{
		URL:   "https://github.com/acme/payments/issues/12",
		Label: "acme/payments#12",
	}, OriginFromIntakeRootEvent(event))
}

func TestOriginFromIntakePayload_MissingURLReturnsNil(t *testing.T) {
	assert.Nil(t, OriginFromIntakePayload(map[string]any{
		"issue": map[string]any{"title": "No URL"},
	}))
}

// GitHub App issue webhooks put installation.html_url (and repository /
// sender html_url) as siblings of issue. Walking those first would stamp
// the App installation URL as origin, and a later lookup by
// issue.html_url would never find the task.
func TestOriginFromIntakePayload_PrefersGitHubIssueOverAppInstallation(t *testing.T) {
	origin := OriginFromIntakePayload(map[string]any{
		"action": "opened",
		"installation": map[string]any{
			"id":       1,
			"html_url": "https://github.com/organizations/acme/settings/installations/1",
		},
		"issue": map[string]any{
			"html_url": "https://github.com/acme/payments/issues/12",
			"url":      "https://api.github.com/repos/acme/payments/issues/12",
			"title":    "Handle duplicate refunds",
			"user": map[string]any{
				"html_url": "https://github.com/octocat",
			},
		},
		"repository": map[string]any{
			"html_url": "https://github.com/acme/payments",
		},
		"sender": map[string]any{
			"html_url": "https://github.com/octocat",
		},
	})

	assert.Equal(t, &WorkOrderOrigin{
		URL:   "https://github.com/acme/payments/issues/12",
		Label: "acme/payments#12",
	}, origin)
}

// PagerDuty webhooks put agent.html_url before incident alphabetically.
// Origin must be the incident, not the user who triggered it.
func TestOriginFromIntakePayload_PrefersPagerDutyIncidentOverAgent(t *testing.T) {
	origin := OriginFromIntakePayload(map[string]any{
		"agent": map[string]any{
			"html_url": "https://acme.pagerduty.com/users/PLH1HKV",
		},
		"incident": map[string]any{
			"html_url": "https://acme.pagerduty.com/incidents/PGR0VU2",
			"title":    "A little bump in the road",
		},
	})

	assert.Equal(t, &WorkOrderOrigin{
		URL:   "https://acme.pagerduty.com/incidents/PGR0VU2",
		Label: "PGR0VU2",
	}, origin)
}

func TestOriginLabelFromURL(t *testing.T) {
	assert.Equal(t, "acme/payments#12", OriginLabelFromURL("https://github.com/acme/payments/issues/12"))
	assert.Equal(t, "acme/payments#8", OriginLabelFromURL("https://github.com/acme/payments/pull/8"))
	assert.Equal(t, "7670162495", OriginLabelFromURL("https://example.com/issues/7670162495/"))
	assert.Equal(t, "P123ABC", OriginLabelFromURL("https://acme.example.com/incidents/P123ABC"))
	assert.Equal(t, "1", OriginLabelFromURL("https://example.com/item/1"))
}
