package jira

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestIsJiraAttachmentURL(t *testing.T) {
	t.Parallel()

	content := testProxyURL("/rest/api/3/attachment/content/10001")
	thumbnail := testProxyURL("/rest/api/3/attachment/thumbnail/10001")
	assert.True(t, IsJiraAttachmentURL(content))
	assert.True(t, IsJiraAttachmentURL(thumbnail))
	assert.True(t, IsJiraAttachmentURL("https://acme.atlassian.net/rest/api/3/attachment/content/10001"))
	assert.False(t, IsJiraAttachmentURL("https://example.com/rest/api/3/attachment/content/10001"))
	assert.False(t, IsJiraAttachmentURL(testProxyURL("/rest/api/3/issue/ENG-1")))
}

func TestPlanIssueDownloadsReplacesDescriptionURLAndAppendsNewFile(t *testing.T) {
	t.Parallel()

	content := testProxyURL("/rest/api/3/attachment/content/10001")
	description := "See ![shot](" + content + ")"
	planned := planIssueDownloads(description, []IssueAttachment{
		{ID: "10001", Filename: "shot.png", MIMEType: "image/png", ContentURL: content},
		{ID: "10002", Filename: "notes.pdf", MIMEType: "application/pdf", ContentURL: testProxyURL("/rest/api/3/attachment/content/10002")},
	})

	require.Len(t, planned, 2)
	assert.Equal(t, "10001", planned[0].attachmentID)
	assert.Equal(t, "shot.png", planned[0].name)
	assert.Equal(t, []string{content}, planned[0].replaceURLs)
	assert.Equal(t, "10002", planned[1].attachmentID)
	assert.Equal(t, "notes.pdf", planned[1].name)
	assert.Empty(t, planned[1].replaceURLs)
}

func TestClientIssueFilesDownloadsWithOAuth(t *testing.T) {
	contentURL := testProxyURL("/rest/api/3/attachment/content/10001")
	httpContext := &contexts.HTTPContext{Responses: []*http.Response{
		{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body: io.NopCloser(strings.NewReader(`{
				"id":"10001",
				"key":"ENG-1",
				"fields":{
					"attachment":[
						{"id":"10001","filename":"notes.pdf","mimeType":"application/pdf","content":"` + contentURL + `"},
						{"id":"10002","filename":"gone.pdf","mimeType":"application/pdf","content":"` + testProxyURL("/rest/api/3/attachment/content/10002") + `"}
					]
				}
			}`)),
		},
		{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/pdf"}},
			Body:       io.NopCloser(strings.NewReader("pdf-bytes")),
		},
		{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/pdf"}},
			Body:       io.NopCloser(strings.NewReader("gone-bytes")),
		},
	}}

	client, err := NewClient(httpContext, newAuthorizedIntegration())
	require.NoError(t, err)

	files, err := client.IssueFiles(t.Context(), "ENG-1", "No files in the text.")
	require.NoError(t, err)
	require.Len(t, files, 2)
	assert.Equal(t, "notes.pdf", files[0].Name)
	assert.Equal(t, "application/pdf", files[0].ContentType)
	assert.Equal(t, []byte("pdf-bytes"), files[0].Body)
	assert.Empty(t, files[0].ReplaceURLs)
	assert.Equal(t, "gone.pdf", files[1].Name)
	assert.Equal(t, []byte("gone-bytes"), files[1].Body)

	require.Len(t, httpContext.Requests, 3)
	assert.Contains(t, httpContext.Requests[0].URL.String(), "/rest/api/3/issue/ENG-1")
	assert.Contains(t, httpContext.Requests[0].URL.RawQuery, "fields=attachment")
	assert.Equal(t, "Bearer "+testAccessToken, httpContext.Requests[1].Header.Get("Authorization"))
	assert.Contains(t, httpContext.Requests[1].URL.String(), "/rest/api/3/attachment/content/10001")
}
