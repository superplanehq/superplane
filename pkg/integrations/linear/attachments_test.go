package linear

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test__IsLinearUploadURL(t *testing.T) {
	assert.True(t, IsLinearUploadURL("https://uploads.linear.app/file/abc/shot.png"))
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

func Test__LinkSectionMarkdown(t *testing.T) {
	section := LinkSectionMarkdown([]IssueLink{{Title: "Pull request", URL: "https://github.com/acme/repo/pull/1"}})
	assert.Contains(t, section, "## Attachments")
	assert.Contains(t, section, "[Pull request](https://github.com/acme/repo/pull/1)")
}
