package runner

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHostedVideoAttachmentsUseAllowlist(t *testing.T) {
	t.Parallel()

	youtube := "https://www.youtube.com/watch?v=dQw4w9WgXcQ"
	text := "See ![Demo](" + youtube + ")\n\n" +
		"Also ![Short](https://www.youtube.com/shorts/dQw4w9WgXcQ)\n\n" +
		"[docs](https://vimeo.com/987654321)\n\n" +
		"![other](https://example.com/clip.mp4)\n\n" +
		"![insecure](http://www.youtube.com/watch?v=dQw4w9WgXcQ)\n\n" +
		"![vimeo](https://vimeo.com/123456789)\n\n" +
		"![loom](https://www.loom.com/share/0123456789abcdef0123456789abcdef)\n\n" +
		"![shot](https://cln.sh/abcd1234)"

	got := HostedVideoAttachments(text)
	require.Len(t, got, 5)
	assert.Equal(t, HostedVideoKind, got[0].Kind)
	assert.Equal(t, youtube, got[0].URL)
	assert.Equal(t, "youtube-dQw4w9WgXcQ", got[0].Filename)
	assert.Equal(t, "vimeo-123456789", got[2].Filename)
	assert.Equal(t, "loom-0123456789abcdef0123456789abcdef", got[3].Filename)
	assert.Equal(t, "cleanshot-abcd1234", got[4].Filename)
}

func TestHostedVideoManifestSetsKindAndDurationPolicy(t *testing.T) {
	t.Parallel()

	body := AttachmentManifestJSON(HostedVideoAttachments("![Demo](https://youtu.be/dQw4w9WgXcQ)"))
	assert.Contains(t, body, `"kind": "hosted_video"`)
	assert.Contains(t, body, `"hosted_video_max_duration_seconds": 300`)
	assert.Contains(t, body, `"max_duration_seconds": 900`)
	assert.Contains(t, body, `"hosted_video_max_bytes": 268435456`)
}

func TestDispatchRewriteCollectsHostedVideosWithoutStoredFiles(t *testing.T) {
	t.Parallel()

	rewriter := &dispatchFileRewriter{}
	text := "![Demo](https://www.youtube-nocookie.com/embed/dQw4w9WgXcQ)"
	got, err := rewriter.Rewrite(text)
	require.NoError(t, err)
	assert.Equal(t, text, got)
	attachments := rewriter.Attachments()
	require.Len(t, attachments, 1)
	assert.Equal(t, HostedVideoKind, attachments[0].Kind)
	assert.Equal(t, "https://www.youtube-nocookie.com/embed/dQw4w9WgXcQ", attachments[0].URL)
}
