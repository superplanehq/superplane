package productive

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestTaskIDFromEventData(t *testing.T) {
	t.Parallel()

	taskID, ok := TaskIDFromEventData(map[string]any{
		"type": TaskPayloadType,
		"data": map[string]any{
			"meta": map[string]any{"event": TaskCreatedEvent},
			"data": map[string]any{"id": "20305431", "type": "tasks"},
		},
	})
	require.True(t, ok)
	assert.Equal(t, "20305431", taskID)

	_, ok = TaskIDFromEventData(map[string]any{
		"type": "github.issue",
		"data": map[string]any{"data": map[string]any{"id": "12"}},
	})
	assert.False(t, ok)
}

func TestIsProductiveFileURL(t *testing.T) {
	t.Parallel()

	assert.True(t, IsProductiveFileURL("https://files.productive.io/attachments/files/1/original/shot.png?1776398568"))
	assert.True(t, IsProductiveFileURL("https://files-test.productive.io/attachments/files/000/000/001/original/img.png"))
	assert.False(t, IsProductiveFileURL("http://files.productive.io/attachments/files/1/original/shot.png"))
	assert.False(t, IsProductiveFileURL("https://api.productive.io/api/v2/attachments/1"))
	assert.False(t, IsProductiveFileURL("https://example.com/attachments/files/1/original/img.png"))
}

func TestPlanTaskDownloadsReplacesDescriptionURLAndAppendsNewFile(t *testing.T) {
	t.Parallel()

	inline := "https://files.productive.io/attachments/files/1/original/shot.png?1"
	description := `<p>See <img src="` + inline + `"></p>`
	planned := planTaskDownloads(description, []Attachment{
		{Name: "shot.png", ContentType: "image/png", URL: "https://files.productive.io/attachments/files/1/original/shot.png?2"},
		{Name: "notes.pdf", ContentType: "application/pdf", URL: "https://files.productive.io/attachments/files/2/original/notes.pdf"},
	})

	require.Len(t, planned, 2)
	assert.Equal(t, "shot.png", planned[0].name)
	assert.Equal(t, []string{inline}, planned[0].replaceURLs)
	assert.Equal(t, "notes.pdf", planned[1].name)
	assert.Empty(t, planned[1].replaceURLs)
}

func TestPlanTaskDownloadsMatchesDescriptionURLWithDifferentQuery(t *testing.T) {
	t.Parallel()

	descriptionURL := "https://files.productive.io/attachments/files/1/original/notes.pdf?token=abc"
	planned := planTaskDownloads(
		`<a href="`+descriptionURL+`">notes</a>`,
		[]Attachment{
			{
				Name:        "notes.pdf",
				ContentType: "application/pdf",
				URL:         "https://files.productive.io/attachments/files/1/original/notes.pdf?1776398568",
			},
		},
	)

	require.Len(t, planned, 1)
	assert.Equal(t, []string{descriptionURL}, planned[0].replaceURLs)
}

func TestClientTaskFilesDownloadsWithToken(t *testing.T) {
	fileURL := "https://files.productive.io/attachments/files/1/original/notes.pdf"
	httpContext := &contexts.HTTPContext{Responses: []*http.Response{
		jsonResponse(`{"data":[
			{"id":"1","type":"attachments","attributes":{"name":"notes.pdf","content_type":"application/pdf","url":"` + fileURL + `","deleted_at":null}},
			{"id":"2","type":"attachments","attributes":{"name":"gone.pdf","content_type":"application/pdf","url":"` + fileURL + `","deleted_at":"2026-04-17T06:02:48Z"}}
		]}`),
		{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/pdf"}},
			Body:       io.NopCloser(strings.NewReader("pdf-bytes")),
		},
	}}

	files, err := testClient(t, httpContext).TaskFiles(t.Context(), "20305431", "No files in the text.")
	require.NoError(t, err)
	require.Len(t, files, 1)
	assert.Equal(t, "notes.pdf", files[0].Name)
	assert.Equal(t, "application/pdf", files[0].ContentType)
	assert.Equal(t, []byte("pdf-bytes"), files[0].Body)
	assert.Empty(t, files[0].ReplaceURLs)

	require.Len(t, httpContext.Requests, 2)
	assert.Contains(t, httpContext.Requests[0].URL.String(), "filter%5Btask_id%5D=20305431")
	assert.Equal(t, "token-1", httpContext.Requests[1].URL.Query().Get("token"))
	assert.NotContains(t, files[0].Name, "token-1")
}
