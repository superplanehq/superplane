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

func TestParseGitHubOrigin(t *testing.T) {
	t.Run("parses issue URL", func(t *testing.T) {
		owner, repo, kind, number, ok := ParseGitHubOrigin("https://github.com/acme/payments/issues/12")
		assert.True(t, ok)
		assert.Equal(t, "acme", owner)
		assert.Equal(t, "payments", repo)
		assert.Equal(t, "issues", kind)
		assert.Equal(t, "12", number)
	})

	t.Run("parses pull request URL", func(t *testing.T) {
		owner, repo, kind, number, ok := ParseGitHubOrigin("https://github.com/acme/payments/pull/8")
		assert.True(t, ok)
		assert.Equal(t, "acme", owner)
		assert.Equal(t, "payments", repo)
		assert.Equal(t, "pull", kind)
		assert.Equal(t, "8", number)
	})

	t.Run("rejects non-GitHub URL", func(t *testing.T) {
		_, _, _, _, ok := ParseGitHubOrigin("https://example.com/issues/12")
		assert.False(t, ok)
	})

	t.Run("rejects missing path segments", func(t *testing.T) {
		_, _, _, _, ok := ParseGitHubOrigin("https://github.com/acme/payments")
		assert.False(t, ok)
	})

	t.Run("rejects unsupported kind", func(t *testing.T) {
		_, _, _, _, ok := ParseGitHubOrigin("https://github.com/acme/payments/discussions/12")
		assert.False(t, ok)
	})

	t.Run("rejects empty number", func(t *testing.T) {
		_, _, _, _, ok := ParseGitHubOrigin("https://github.com/acme/payments/issues/")
		assert.False(t, ok)
	})
}

func TestOriginLabelFromURL(t *testing.T) {
	assert.Equal(t, "acme/payments#12", OriginLabelFromURL("https://github.com/acme/payments/issues/12"))
	assert.Equal(t, "acme/payments#8", OriginLabelFromURL("https://github.com/acme/payments/pull/8"))
	assert.Equal(t, "7670162495", OriginLabelFromURL("https://example.com/issues/7670162495/"))
	assert.Equal(t, "P123ABC", OriginLabelFromURL("https://acme.example.com/incidents/P123ABC"))
	assert.Equal(t, "1", OriginLabelFromURL("https://example.com/item/1"))
}
