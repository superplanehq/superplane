package prbody

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func TestUpdateBodySection(t *testing.T) {
	const StartMarker = "<!-- superplane:merge-confidence:start -->"
	const EndMarker = "<!-- superplane:merge-confidence:end -->"
	const section = "<!-- superplane:merge-confidence:start -->\n## SuperPlane merge confidence\nnew\n<!-- superplane:merge-confidence:end -->"
	const old = "<!-- superplane:merge-confidence:start -->old<!-- superplane:merge-confidence:end -->"
	const greptile = "<!-- greptile_comment -->\r\nConfidence: 4/5\r\n<!-- /greptile_comment -->"
	for _, tc := range []struct {
		name, body, want string
		invalid          bool
	}{
		{name: "empty", want: section},
		{name: "append", body: "Original\r\n" + greptile, want: "Original\r\n" + greptile + "\n\n" + section},
		{name: "replace preserves bytes", body: " \r\n" + greptile + "\r\n" + old + "\n trailing \t", want: " \r\n" + greptile + "\r\n" + section + "\n trailing \t"},
		{name: "unchanged", body: section, want: section},
		{name: "other owned sections", body: "<!-- superplane:notes:start -->notes<!-- superplane:notes:end -->" + old, want: "<!-- superplane:notes:start -->notes<!-- superplane:notes:end -->" + section},
		{name: "missing end", body: StartMarker, invalid: true},
		{name: "missing start", body: EndMarker, invalid: true},
		{name: "reversed", body: EndMarker + StartMarker, invalid: true},
		{name: "duplicate", body: old + old, invalid: true},
		{name: "duplicate start", body: StartMarker + old, invalid: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := UpdateBodySection(tc.body, "merge-confidence", "## SuperPlane merge confidence\nnew")
			if tc.invalid {
				require.Error(t, err)
				assert.Equal(t, tc.body, got)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestUpdateBodySectionRejectsUnsafeKeysAndOwnedMarkers(t *testing.T) {
	for _, key := range []string{"", "Mixed", "has space", "a--b", "a-->b", "a\nb", strings.Repeat("a", 65)} {
		got, err := UpdateBodySection("original", key, "arbitrary")
		require.Error(t, err)
		assert.Equal(t, "original", got)
	}
	got, err := UpdateBodySection("original", "notes", "<!-- superplane:notes:end -->")
	require.Error(t, err)
	assert.Equal(t, "original", got)
}
