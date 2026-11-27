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

func TestClientIssueFilesImportsOnlySourceAttachments(t *testing.T) {
	contentURL := "https://example.atlassian.net/rest/api/3/attachment/content/10001"
	thumbnailURL := "https://example.atlassian.net/rest/api/3/attachment/thumbnail/10001"
	foreignURL := "https://example.atlassian.net/rest/api/3/attachment/content/90001"
	foreignDescription := "[file](" + foreignURL + ")"
	issueURL := testProxyURL("/rest/api/3/issue/PUBLIC-1")
	authorizedURL := testProxyURL("/rest/api/3/attachment/content/10001")
	foreignContentURL := testProxyURL("/rest/api/3/attachment/content/90001")

	for _, tt := range []struct {
		name        string
		fields      string
		description string
		wantFiles   []IssueFile
	}{
		{
			name:        "empty attachment list",
			fields:      `{"attachment":[]}`,
			description: foreignDescription,
		},
		{
			name:        "missing attachment list",
			fields:      `{}`,
			description: foreignDescription,
		},
		{
			name:        "attachment without an ID",
			fields:      `{"attachment":[{"filename":"foreign.pdf","content":"` + foreignURL + `"}]}`,
			description: foreignDescription,
		},
		{
			name: "authorized content and thumbnail with foreign link",
			fields: `{"attachment":[{"id":" 10001 ","filename":"notes.pdf","mimeType":"application/pdf",` +
				`"content":"` + authorizedURL + `","thumbnail":"` + testProxyURL("/rest/api/3/attachment/thumbnail/10001") + `"}]}`,
			description: foreignDescription + " [notes](" + contentURL + ") ![preview](" + thumbnailURL + ") [notes again](" + contentURL + ")",
			wantFiles: []IssueFile{{
				Name:        "notes.pdf",
				ContentType: "application/pdf",
				Body:        []byte("authorized-pdf-bytes"),
				ReplaceURLs: []string{authorizedURL, contentURL, thumbnailURL},
			}},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			httpContext := &contexts.HTTPContext{ResponsesByPath: map[string][]*http.Response{
				strings.TrimPrefix(issueURL, "https://api.atlassian.com"): {{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"key":"PUBLIC-1","fields":` + tt.fields + `}`)),
				}},
				strings.TrimPrefix(authorizedURL, "https://api.atlassian.com"): {{
					StatusCode: http.StatusOK,
					Header:     http.Header{"Content-Type": []string{"application/pdf"}},
					Body:       io.NopCloser(strings.NewReader("authorized-pdf-bytes")),
				}},
				// A successful foreign response proves rejection happens before provider access.
				strings.TrimPrefix(foreignContentURL, "https://api.atlassian.com"): {{
					StatusCode: http.StatusOK,
					Header:     http.Header{"Content-Type": []string{"application/pdf"}},
					Body:       io.NopCloser(strings.NewReader("private-pdf-bytes")),
				}},
			}}
			client, err := NewClient(httpContext, newAuthorizedIntegration())
			require.NoError(t, err)

			files, err := client.IssueFiles(t.Context(), "PUBLIC-1", tt.description)
			require.NoError(t, err)
			assert.Equal(t, tt.wantFiles, files)
			requestURLs := make([]string, 0, len(httpContext.Requests))
			for _, request := range httpContext.Requests {
				requestURLs = append(requestURLs, request.URL.Scheme+"://"+request.URL.Host+request.URL.Path)
				assert.Equal(t, "Bearer "+testAccessToken, request.Header.Get("Authorization"))
			}
			wantRequestURLs := []string{issueURL}
			if len(tt.wantFiles) > 0 {
				wantRequestURLs = append(wantRequestURLs, authorizedURL)
			}
			assert.Equal(t, wantRequestURLs, requestURLs)
			require.NotEmpty(t, httpContext.Requests)
			assert.Equal(t, "attachment", httpContext.Requests[0].URL.Query().Get("fields"))
		})
	}
}
