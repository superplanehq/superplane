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

func TestOriginLabelFromURL(t *testing.T) {
	assert.Equal(t, "acme/payments#12", OriginLabelFromURL("https://github.com/acme/payments/issues/12"))
	assert.Equal(t, "acme/payments#8", OriginLabelFromURL("https://github.com/acme/payments/pull/8"))
	assert.Equal(t, "7670162495", OriginLabelFromURL("https://example.com/issues/7670162495/"))
	assert.Equal(t, "P123ABC", OriginLabelFromURL("https://acme.example.com/incidents/P123ABC"))
	assert.Equal(t, "1", OriginLabelFromURL("https://example.com/item/1"))
}

func TestParseGitHubIssueOrigin(t *testing.T) {
	ref, ok := ParseGitHubIssueOrigin("https://github.com/acme/payments/issues/12")
	assert.True(t, ok)
	assert.Equal(t, GitHubIssueOrigin{Repository: "acme/payments", Number: 12}, ref)

	ref, ok = ParseGitHubIssueOrigin("https://github.com/acme/payments/issues/12#issuecomment-1")
	assert.True(t, ok)
	assert.Equal(t, GitHubIssueOrigin{Repository: "acme/payments", Number: 12}, ref)

	_, ok = ParseGitHubIssueOrigin("https://github.com/acme/payments/pull/8")
	assert.False(t, ok)

	_, ok = ParseGitHubIssueOrigin("https://app.productive.io/12345/tasks/91")
	assert.False(t, ok)

	_, ok = ParseGitHubIssueOrigin("")
	assert.False(t, ok)
}
