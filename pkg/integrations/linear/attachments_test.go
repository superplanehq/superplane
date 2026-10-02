package linear

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__IsLinearUploadURL(t *testing.T) {
	assert.True(t, IsLinearUploadURL("https://uploads.linear.app/file/abc/shot.png"))
	assert.False(t, IsLinearUploadURL("http://uploads.linear.app/file/abc/shot.png"))
	assert.False(t, IsLinearUploadURL("https://github.com/acme/repo/pull/1"))
}

func Test__planLinearDownloads(t *testing.T) {
	description := "See https://uploads.linear.app/file/abc/shot.png and the spec."
	planned := planLinearDownloads(description, []Attachment{
		{Title: "shot.png", URL: "https://uploads.linear.app/file/abc/shot.png"},
		{Title: "Pull request", URL: "https://github.com/acme/repo/pull/1"},
	})

	require.Len(t, planned, 1)
	assert.Equal(t, "shot.png", planned[0].name)
	assert.Equal(t, []string{"https://uploads.linear.app/file/abc/shot.png"}, planned[0].replaceURLs)
}

func Test__IssueFiles__DoesNotSendTheTokenOverHTTP(t *testing.T) {
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			jsonResponse(`{"data":{"issue":{"attachments":{"nodes":[{"id":"a1","title":"shot.png","url":"http://uploads.linear.app/file/abc/shot.png"}]}}}}`),
		},
	}
	client, err := NewClient(httpContext, newAuthorizedIntegration())
	require.NoError(t, err)

	rawURL := "http://uploads.linear.app/file/abc/shot.png"
	files, links, err := client.IssueFiles(context.Background(), "issue-1", "See "+rawURL)
	require.NoError(t, err)
	assert.Empty(t, files)
	require.Len(t, links, 1)
	assert.Equal(t, rawURL, links[0].URL)

	for _, request := range httpContext.Requests {
		if request.URL != nil && request.URL.Host == linearUploadHost {
			t.Fatalf("sent credentials to %s", request.URL)
		}
	}
}

func Test__IssueFiles__SendsTheTokenOverHTTPS(t *testing.T) {
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			jsonResponse(`{"data":{"issue":{"attachments":{"nodes":[]}}}}`),
			{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"image/png"}},
				Body:       io.NopCloser(strings.NewReader("png-bytes")),
			},
		},
	}
	client, err := NewClient(httpContext, newAuthorizedIntegration())
	require.NoError(t, err)

	rawURL := "https://uploads.linear.app/file/abc/shot.png"
	files, _, err := client.IssueFiles(context.Background(), "issue-1", "See ![shot]("+rawURL+")")
	require.NoError(t, err)
	require.Len(t, files, 1)
	assert.Equal(t, "png-bytes", string(files[0].Body))
	require.Len(t, httpContext.Requests, 2)
	download := httpContext.Requests[1]
	assert.Equal(t, "https", download.URL.Scheme)
	assert.Equal(t, linearUploadHost, download.URL.Host)
	assert.Equal(t, "Bearer "+testAccessToken, download.Header.Get("Authorization"))
}

func Test__LinkSectionMarkdown(t *testing.T) {
	section := LinkSectionMarkdown([]IssueLink{{Title: "Pull request", URL: "https://github.com/acme/repo/pull/1"}})
	assert.Contains(t, section, "## Attachments")
	assert.Contains(t, section, "[Pull request](https://github.com/acme/repo/pull/1)")
}
