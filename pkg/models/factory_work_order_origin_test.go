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

func TestOriginFromIntakePayload_PrefersJiraBrowseURLOverProxySelf(t *testing.T) {
	origin := OriginFromIntakePayload(map[string]any{
		"action": "created",
		"url":    "https://acme.atlassian.net/browse/ENG-42",
		"issue": map[string]any{
			"key":  "ENG-42",
			"self": "https://api.atlassian.com/ex/jira/cloud-id/rest/api/3/issue/10001",
			"fields": map[string]any{
				"self": "https://api.atlassian.com/ex/jira/cloud-id/rest/api/3/issue/10001",
			},
		},
	})

	assert.Equal(t, &WorkOrderOrigin{
		URL:   "https://acme.atlassian.net/browse/ENG-42",
		Label: "ENG-42",
	}, origin)
}

func TestOriginFromIntakePayload_MissingURLReturnsNil(t *testing.T) {
	assert.Nil(t, OriginFromIntakePayload(map[string]any{
		"issue": map[string]any{"title": "No URL"},
	}))
}

func TestOriginFromIntakePayload_UsesSentryIssueTitle(t *testing.T) {
	origin := OriginFromIntakePayload(map[string]any{
		"data": map[string]any{
			"issue": map[string]any{
				"permalink": "https://acme.sentry.io/issues/7670162495/",
				"title":     "  TypeError: boom  ",
			},
		},
	})

	assert.Equal(t, &WorkOrderOrigin{
		URL:   "https://acme.sentry.io/issues/7670162495/",
		Label: "TypeError: boom",
	}, origin)
}

func TestOriginFromIntakePayload_UsesTopLevelSentryIssueTitle(t *testing.T) {
	origin := OriginFromIntakePayload(map[string]any{
		"issue": map[string]any{
			"permalink": "https://sentry.io/issues/123/",
			"title":     "Broken deploy",
		},
	})

	assert.Equal(t, &WorkOrderOrigin{
		URL:   "https://sentry.io/issues/123/",
		Label: "Broken deploy",
	}, origin)
}

func TestOriginFromIntakePayload_FallsBackWhenSentryTitleMissing(t *testing.T) {
	origin := OriginFromIntakePayload(map[string]any{
		"data": map[string]any{
			"issue": map[string]any{
				"permalink": "https://acme.sentry.io/issues/7670162495/",
			},
		},
	})

	assert.Equal(t, &WorkOrderOrigin{
		URL:   "https://acme.sentry.io/issues/7670162495/",
		Label: "7670162495",
	}, origin)
}

func TestFactoryWorkOrderOrigin_KeepsStoredLabel(t *testing.T) {
	originURL := "https://acme.sentry.io/issues/123/"
	label := "123"
	order := &FactoryWorkOrder{OriginURL: &originURL, OriginLabel: &label}

	assert.Equal(t, &WorkOrderOrigin{URL: originURL, Label: label}, order.Origin())
}

func TestOriginLabelFromURL(t *testing.T) {
	assert.Equal(t, "acme/payments#12", OriginLabelFromURL("https://github.com/acme/payments/issues/12"))
	assert.Equal(t, "acme/payments#8", OriginLabelFromURL("https://github.com/acme/payments/pull/8"))
	assert.Equal(t, "7670162495", OriginLabelFromURL("https://example.com/issues/7670162495/"))
	assert.Equal(t, "P123ABC", OriginLabelFromURL("https://acme.example.com/incidents/P123ABC"))
	assert.Equal(t, "1", OriginLabelFromURL("https://example.com/item/1"))
	assert.Equal(t, "ENG-42", OriginLabelFromURL("https://acme.atlassian.net/browse/ENG-42"))
}

func TestOriginLabelFromIntake(t *testing.T) {
	assert.Equal(
		t,
		"TypeError: boom",
		OriginLabelFromIntake("https://acme.sentry.io/issues/123/", "  TypeError: boom  "),
	)
	assert.Equal(t, "123", OriginLabelFromIntake("https://acme.sentry.io/issues/123/", "  "))
	assert.Equal(
		t,
		"acme/payments#12",
		OriginLabelFromIntake("https://github.com/acme/payments/issues/12", "Handle duplicate refunds"),
	)
}
