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

func TestOriginFromIntakePayload_LabelsDependabotAlert(t *testing.T) {
	origin := OriginFromIntakePayload(map[string]any{
		"alert": map[string]any{
			"html_url": "https://github.com/acme/payments/security/dependabot/7",
		},
	})

	assert.Equal(t, &WorkOrderOrigin{
		URL:   "https://github.com/acme/payments/security/dependabot/7",
		Label: "acme/payments dependabot #7",
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

func TestOriginFromIntakePayload_UsesSentryIssueTitleAsLabel(t *testing.T) {
	origin := OriginFromIntakePayload(map[string]any{
		"resource": "issue",
		"action":   "created",
		"data": map[string]any{
			"issue": map[string]any{
				"id":        "123",
				"title":     "  Error #1:\nThis is a test error!  ",
				"permalink": "https://acme.sentry.io/issues/123/",
				"web_url":   "https://acme.sentry.io/issues/123/",
			},
		},
	})

	assert.Equal(t, &WorkOrderOrigin{
		URL:   "https://acme.sentry.io/issues/123/",
		Label: "Error #1: This is a test error!",
	}, origin)
}

func TestOriginFromIntakeRootEvent_UsesSentryIssueTitleAsLabel(t *testing.T) {
	event := &CanvasEvent{Data: NewJSONValue(map[string]any{
		"type": "sentry.issue",
		"data": map[string]any{
			"resource": "issue",
			"action":   "created",
			"data": map[string]any{
				"issue": map[string]any{
					"id":        "123",
					"title":     "Error #1: This is a test error!",
					"permalink": "https://acme.sentry.io/issues/123/",
				},
			},
		},
	})}

	assert.Equal(t, &WorkOrderOrigin{
		URL:   "https://acme.sentry.io/issues/123/",
		Label: "Error #1: This is a test error!",
	}, OriginFromIntakeRootEvent(event))
}

func TestOriginFromIntakePayload_UsesDatadogIssueTitleAsLabel(t *testing.T) {
	const issueID = "da226b38-baac-11f1-bad1-da7ad0900005"
	cases := []struct {
		name string
		link string
	}{
		{name: "eu site", link: "https://app.datadoghq.eu/error-tracking/issue/" + issueID},
		{name: "us3 site", link: "https://app.us3.datadoghq.com/error-tracking/issue/" + issueID},
		{name: "gov site", link: "https://app.ddog-gov.com/error-tracking/issue/" + issueID + "?from_ts=1"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			origin := OriginFromIntakePayload(map[string]any{
				"event_type": "error_tracking_alert",
				"title":      "  TimeoutError:\ncheckout timed out  ",
				"link":       tc.link,
				"body":       "See the issue in Datadog.",
			})

			assert.Equal(t, &WorkOrderOrigin{
				URL:   tc.link,
				Label: "TimeoutError: checkout timed out",
			}, origin)
		})
	}
}

func TestOriginFromIntakeRootEvent_UsesDatadogIssueTitleAsLabel(t *testing.T) {
	const issueID = "da226b38-baac-11f1-bad1-da7ad0900005"
	link := "https://app.datadoghq.com/error-tracking/issue/" + issueID
	event := &CanvasEvent{Data: NewJSONValue(map[string]any{
		"type": "datadog.errorTrackingAlert",
		"data": map[string]any{
			"title": "TimeoutError: checkout timed out",
			"link":  link,
		},
	})}

	assert.Equal(t, &WorkOrderOrigin{
		URL:   link,
		Label: "TimeoutError: checkout timed out",
	}, OriginFromIntakeRootEvent(event))
}

func TestOriginFromIntakePayload_FallsBackWhenDatadogTitleIsMissing(t *testing.T) {
	const issueID = "da226b38-baac-11f1-bad1-da7ad0900005"
	link := "https://app.datadoghq.com/error-tracking/issue/" + issueID + "/"

	t.Run("missing title", func(t *testing.T) {
		origin := OriginFromIntakePayload(map[string]any{
			"link": link,
		})

		assert.Equal(t, &WorkOrderOrigin{
			URL:   link,
			Label: issueID,
		}, origin)
	})

	t.Run("empty title", func(t *testing.T) {
		origin := OriginFromIntakePayload(map[string]any{
			"title": "  \n  ",
			"link":  link,
		})

		assert.Equal(t, &WorkOrderOrigin{
			URL:   link,
			Label: issueID,
		}, origin)
	})
}

func TestOriginFromIntakePayload_IgnoresTitleOutsideDatadogIssuePage(t *testing.T) {
	origin := OriginFromIntakePayload(map[string]any{
		"title": "TimeoutError: checkout timed out",
		"link":  "https://app.datadoghq.com/monitors/98765",
	})
	assert.Equal(t, &WorkOrderOrigin{
		URL:   "https://app.datadoghq.com/monitors/98765",
		Label: "98765",
	}, origin)

	origin = OriginFromIntakePayload(map[string]any{
		"title": "TimeoutError: checkout timed out",
		"link":  "https://example.com/error-tracking/issue/abc",
	})
	assert.Equal(t, &WorkOrderOrigin{
		URL:   "https://example.com/error-tracking/issue/abc",
		Label: "abc",
	}, origin)
}

func TestFactoryWorkOrder_OriginShowsTaskTitleForLegacyDatadogIssueID(t *testing.T) {
	const issueID = "da226b38-baac-11f1-bad1-da7ad0900005"
	rawURL := "https://app.datadoghq.eu/error-tracking/issue/" + issueID
	title := "TimeoutError: checkout timed out"

	t.Run("saved label is the issue id", func(t *testing.T) {
		saved := issueID
		order := &FactoryWorkOrder{Title: title, OriginURL: &rawURL, OriginLabel: &saved}

		origin := order.Origin()
		assert.Equal(t, &WorkOrderOrigin{URL: rawURL, Label: title}, origin)
		assert.Equal(t, issueID, *order.OriginLabel)
	})

	t.Run("saved label is empty", func(t *testing.T) {
		saved := "  "
		order := &FactoryWorkOrder{Title: title, OriginURL: &rawURL, OriginLabel: &saved}

		assert.Equal(t, title, order.Origin().Label)
		assert.Equal(t, "  ", *order.OriginLabel)
	})

	t.Run("empty task title keeps the issue id", func(t *testing.T) {
		saved := issueID
		order := &FactoryWorkOrder{Title: "  \n ", OriginURL: &rawURL, OriginLabel: &saved}

		assert.Equal(t, issueID, order.Origin().Label)
	})
}

func TestFactoryWorkOrder_OriginKeepsSavedDatadogTitle(t *testing.T) {
	const issueID = "da226b38-baac-11f1-bad1-da7ad0900005"
	rawURL := "https://app.datadoghq.eu/error-tracking/issue/" + issueID
	saved := "TimeoutError: checkout timed out"
	order := &FactoryWorkOrder{
		Title:       "Renamed task",
		OriginURL:   &rawURL,
		OriginLabel: &saved,
	}

	assert.Equal(t, saved, order.Origin().Label)
}

func TestFactoryWorkOrder_OriginDoesNotReplaceOtherSourceLabels(t *testing.T) {
	title := "TimeoutError: checkout timed out"

	t.Run("github", func(t *testing.T) {
		rawURL := "https://github.com/acme/payments/issues/12"
		saved := "acme/payments#12"
		order := &FactoryWorkOrder{Title: title, OriginURL: &rawURL, OriginLabel: &saved}

		assert.Equal(t, saved, order.Origin().Label)
	})

	t.Run("jira", func(t *testing.T) {
		rawURL := "https://acme.atlassian.net/browse/ENG-42"
		saved := "ENG-42"
		order := &FactoryWorkOrder{Title: title, OriginURL: &rawURL, OriginLabel: &saved}

		assert.Equal(t, saved, order.Origin().Label)
	})

	t.Run("sentry", func(t *testing.T) {
		rawURL := "https://acme.sentry.io/issues/123/"
		saved := "Error #1: This is a test error!"
		order := &FactoryWorkOrder{Title: title, OriginURL: &rawURL, OriginLabel: &saved}

		assert.Equal(t, saved, order.Origin().Label)
	})

	t.Run("datadog monitor", func(t *testing.T) {
		rawURL := "https://app.datadoghq.com/monitors/98765"
		saved := "98765"
		order := &FactoryWorkOrder{Title: title, OriginURL: &rawURL, OriginLabel: &saved}

		assert.Equal(t, saved, order.Origin().Label)
	})
}

func TestOriginFromIntakePayload_FallsBackWhenSentryTitleIsMissing(t *testing.T) {
	t.Run("missing title", func(t *testing.T) {
		origin := OriginFromIntakePayload(map[string]any{
			"data": map[string]any{
				"issue": map[string]any{
					"id":        "123",
					"permalink": "https://acme.sentry.io/issues/123/",
				},
			},
		})

		assert.Equal(t, &WorkOrderOrigin{
			URL:   "https://acme.sentry.io/issues/123/",
			Label: "123",
		}, origin)
	})

	t.Run("empty title", func(t *testing.T) {
		origin := OriginFromIntakePayload(map[string]any{
			"issue": map[string]any{
				"title":     "  \n  ",
				"permalink": "https://acme.sentry.io/issues/7670162495/",
			},
		})

		assert.Equal(t, &WorkOrderOrigin{
			URL:   "https://acme.sentry.io/issues/7670162495/",
			Label: "7670162495",
		}, origin)
	})
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

func TestOriginLabelFromURL(t *testing.T) {
	assert.Equal(t, "acme/payments#12", OriginLabelFromURL("https://github.com/acme/payments/issues/12"))
	assert.Equal(t, "acme/payments#8", OriginLabelFromURL("https://github.com/acme/payments/pull/8"))
	assert.Equal(t, "7670162495", OriginLabelFromURL("https://example.com/issues/7670162495/"))
	assert.Equal(t, "P123ABC", OriginLabelFromURL("https://acme.example.com/incidents/P123ABC"))
	assert.Equal(t, "1", OriginLabelFromURL("https://example.com/item/1"))
	assert.Equal(t, "ENG-42", OriginLabelFromURL("https://acme.atlassian.net/browse/ENG-42"))
}
