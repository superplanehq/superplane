package blob

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileIDsAndRewrite(t *testing.T) {
	id := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	markdown := "See ![bug](" + FileRef(id) + ") and <img src=\"" + FileRef(id) + "\">"
	ids := FileIDsInMarkdown(markdown)
	require.Equal(t, []uuid.UUID{id}, ids)

	rewritten := RewriteFileRefs(markdown, map[uuid.UUID]string{id: "https://cdn.example/x.png"})
	assert.Contains(t, rewritten, "https://cdn.example/x.png")
	assert.NotContains(t, rewritten, FileRefScheme)
}

func TestGitHubImageURLDetection(t *testing.T) {
	assert.True(t, IsGitHubImageURL("https://private-user-images.githubusercontent.com/1/2.png?jwt=abc"))
	assert.True(t, IsGitHubImageURL("https://github.com/acme/app/assets/1/2"))
	assert.False(t, IsGitHubImageURL("https://example.com/x.png"))
	assert.Equal(
		t,
		[]string{"https://user-images.githubusercontent.com/1.png"},
		HTTPImageURLs("![x](https://user-images.githubusercontent.com/1.png) and [docs](https://example.com/a.pdf)"),
	)
}
