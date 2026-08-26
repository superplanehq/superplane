package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOriginFromIntakePayload_GitHubIssue(t *testing.T) {
	origin := OriginFromIntakePayload(FactoryIntakeSourceGitHubIssues, map[string]any{
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

func TestOriginFromIntakePayload_SentryException(t *testing.T) {
	origin := OriginFromIntakePayload(FactoryIntakeSourceSentryExceptions, map[string]any{
		"data": map[string]any{
			"issue": map[string]any{
				"permalink": "https://superplane.sentry.io/issues/7670162495/",
			},
		},
	})

	assert.Equal(t, &WorkOrderOrigin{
		URL:   "https://superplane.sentry.io/issues/7670162495/",
		Label: "superplane#7670162495",
	}, origin)
}

func TestOriginFromIntakePayload_PagerDutyIncident(t *testing.T) {
	origin := OriginFromIntakePayload(FactoryIntakeSourcePagerDutyIncidents, map[string]any{
		"incident": map[string]any{
			"html_url": "https://acme.pagerduty.com/incidents/P123ABC",
		},
	})

	assert.Equal(t, &WorkOrderOrigin{
		URL:   "https://acme.pagerduty.com/incidents/P123ABC",
		Label: "acme#P123ABC",
	}, origin)
}

func TestOriginFromIntakeRootEvent_PeelsEnvelopeThenReadsURL(t *testing.T) {
	githubEvent := &CanvasEvent{Data: NewJSONValue(map[string]any{
		"type": "github.issue",
		"data": map[string]any{
			"issue": map[string]any{"html_url": "https://github.com/acme/payments/issues/12"},
		},
	})}
	sentryEvent := &CanvasEvent{Data: NewJSONValue(map[string]any{
		"type": "sentry.exception",
		"data": map[string]any{
			"data": map[string]any{
				"issue": map[string]any{"permalink": "https://superplane.sentry.io/issues/7670162495/"},
			},
		},
	})}
	pagerEvent := &CanvasEvent{Data: NewJSONValue(map[string]any{
		"type": "pagerduty.incident",
		"data": map[string]any{
			"incident": map[string]any{"html_url": "https://acme.pagerduty.com/incidents/P123ABC"},
		},
	})}

	assert.Equal(t, &WorkOrderOrigin{
		URL:   "https://github.com/acme/payments/issues/12",
		Label: "acme/payments#12",
	}, OriginFromIntakeRootEvent(FactoryIntakeSourceGitHubIssues, githubEvent))
	assert.Equal(t, &WorkOrderOrigin{
		URL:   "https://superplane.sentry.io/issues/7670162495/",
		Label: "superplane#7670162495",
	}, OriginFromIntakeRootEvent(FactoryIntakeSourceSentryExceptions, sentryEvent))
	assert.Equal(t, &WorkOrderOrigin{
		URL:   "https://acme.pagerduty.com/incidents/P123ABC",
		Label: "acme#P123ABC",
	}, OriginFromIntakeRootEvent(FactoryIntakeSourcePagerDutyIncidents, pagerEvent))
}

func TestOriginFromIntakePayload_MissingURLReturnsNil(t *testing.T) {
	assert.Nil(t, OriginFromIntakePayload(FactoryIntakeSourceGitHubIssues, map[string]any{
		"issue": map[string]any{"title": "No URL"},
	}))
	assert.Nil(t, OriginFromIntakePayload("unknown", map[string]any{
		"issue": map[string]any{"html_url": "https://github.com/acme/app/issues/1"},
	}))
}

func TestOriginLabelFromURL(t *testing.T) {
	assert.Equal(t, "acme/payments#12", OriginLabelFromURL("https://github.com/acme/payments/issues/12"))
	assert.Equal(t, "acme/payments#8", OriginLabelFromURL("https://github.com/acme/payments/pull/8"))
	assert.Equal(t, "superplane#7670162495", OriginLabelFromURL("https://superplane.sentry.io/issues/7670162495/events/abc/"))
	assert.Equal(t, "acme#P123ABC", OriginLabelFromURL("https://acme.pagerduty.com/incidents/P123ABC"))
	assert.Equal(t, "https://example.com/item/1", OriginLabelFromURL("https://example.com/item/1"))
}
