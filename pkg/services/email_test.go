package services

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderNotificationEmailHTML_withURLLabel_omitsDuplicateInlineURL(t *testing.T) {
	repoTemplates := filepath.Join("..", "..", "templates")

	html, err := renderEmailTemplate(repoTemplates, "notification.html", NotificationTemplateData{
		Title:    "Approval needed",
		Body:     "Please review the pending approval.",
		URL:      "https://app.superplane.com/approvals/123",
		URLLabel: "Open approval",
	})
	require.NoError(t, err)

	assert.Contains(t, html, `href="https://app.superplane.com/approvals/123"`)
	assert.Contains(t, html, "Open approval")
	assert.NotContains(t, strings.TrimSpace(html), "https://app.superplane.com/approvals/123</a>")
	assert.NotContains(t, html, ">https://app.superplane.com/approvals/123<")
}

func TestRenderNotificationEmailHTML_withoutURLLabel_keepsInlineURL(t *testing.T) {
	repoTemplates := filepath.Join("..", "..", "templates")

	html, err := renderEmailTemplate(repoTemplates, "notification.html", NotificationTemplateData{
		Title:    "Notice",
		Body:     "Something happened.",
		URL:      "https://example.com/page",
		URLLabel: "",
	})
	require.NoError(t, err)

	assert.Contains(t, html, `href="https://example.com/page"`)
	assert.Contains(t, html, ">https://example.com/page<")
	assert.Contains(t, html, "Open in SuperPlane")
}
