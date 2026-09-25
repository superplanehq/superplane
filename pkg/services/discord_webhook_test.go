package services

import (
	"encoding/json"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiscordWebhookClient_SendSupportFeedbackJSON(t *testing.T) {
	var got map[string]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		require.NoError(t, json.NewDecoder(r.Body).Decode(&got))
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(server.Close)

	client := NewDiscordWebhookClient(server.URL)
	err := client.SendSupportFeedback(SupportFeedback{
		Category:         FeedbackCategoryBug,
		Details:          "The canvas did not load.",
		UserName:         "Ada Lovelace",
		UserEmail:        "ada@example.com",
		OrganizationName: "Acme",
		OrganizationID:   "org-1",
		PagePath:         "/acme/apps/deploy",
	})
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Contains(t, got["content"], "**Bug report**")
	assert.Contains(t, got["content"], "Ada Lovelace <ada@example.com>")
	assert.Contains(t, got["content"], "Acme (org-1)")
	assert.Contains(t, got["content"], "`/acme/apps/deploy`")
	assert.Contains(t, got["content"], "The canvas did not load.")
}

func TestDiscordWebhookClient_SendSupportFeedbackWithFile(t *testing.T) {
	var payload map[string]string
	var filename string
	var fileBytes []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mediaType, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		require.NoError(t, err)
		assert.Equal(t, "multipart/form-data", mediaType)

		reader := multipart.NewReader(r.Body, params["boundary"])
		for {
			part, err := reader.NextPart()
			if err == io.EOF {
				break
			}
			require.NoError(t, err)
			body, err := io.ReadAll(part)
			require.NoError(t, err)
			if part.FormName() == "payload_json" {
				require.NoError(t, json.Unmarshal(body, &payload))
			}
			if part.FormName() == "files[0]" {
				filename = part.FileName()
				fileBytes = body
			}
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(server.Close)

	client := NewDiscordWebhookClient(server.URL)
	err := client.SendSupportFeedback(SupportFeedback{
		Category: FeedbackCategoryFeature,
		Details:  "Add dark mode to the menu.",
		UserName: "Ada",
		Attachment: &SupportFeedbackAttachment{
			Filename:    "menu.png",
			ContentType: "image/png",
			Content:     []byte("png-bytes"),
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "menu.png", filename)
	assert.Equal(t, []byte("png-bytes"), fileBytes)
	assert.Contains(t, payload["content"], "Attachment: menu.png")
}

func TestDiscordWebhookClient_Disabled(t *testing.T) {
	client := NewDiscordWebhookClient("  ")
	assert.False(t, client.Enabled())
	require.NoError(t, client.SendSupportFeedback(SupportFeedback{Category: FeedbackCategoryOther, Details: "hi"}))
}

func TestFormatDiscordFeedbackContentTruncates(t *testing.T) {
	content := FormatDiscordFeedbackContent(SupportFeedback{
		Category: FeedbackCategoryOther,
		Details:  strings.Repeat("x", maxDiscordWebhookContentRunes),
	})
	assert.Equal(t, maxDiscordWebhookContentRunes, len([]rune(content)))
	assert.True(t, strings.HasSuffix(content, "…"))
}
