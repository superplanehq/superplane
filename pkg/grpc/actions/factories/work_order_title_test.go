package factories

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test__titleFromWorkOrderDescription(t *testing.T) {
	const realLine = "Refunds fail on retry."

	tests := []struct {
		name        string
		description string
		want        string
	}{
		{
			name:        "skips an attachment file link and uses the next line",
			description: "[notes.pdf](sp-file://aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee)\n" + realLine,
			want:        realLine,
		},
		{
			name:        "returns empty when the description is only a file link",
			description: "[notes.pdf](sp-file://abc)",
			want:        "",
		},
		{
			name:        "skips a list-wrapped file link",
			description: "- [notes.pdf](sp-file://abc)\n" + realLine,
			want:        realLine,
		},
		{
			name:        "skips a list placeholder and uses the next line",
			description: "- n/a\n" + realLine,
			want:        realLine,
		},
		{
			name:        "skips a star list placeholder",
			description: "* unknown\n" + realLine,
			want:        realLine,
		},
		{
			name:        "skips an ordered list placeholder",
			description: "1. n/a\n" + realLine,
			want:        realLine,
		},
		{
			name:        "skips a task-list placeholder",
			description: "- [ ] n/a\n" + realLine,
			want:        realLine,
		},
		{
			name:        "skips a quoted placeholder inside a list item",
			description: "- \"null\"\n" + realLine,
			want:        realLine,
		},
		{
			name:        "strips a list marker from a real title",
			description: "- " + realLine,
			want:        realLine,
		},
		{
			name:        "strips an ordered list marker from a real title",
			description: "1. " + realLine,
			want:        realLine,
		},
		{
			name:        "keeps a list item that is not a placeholder",
			description: "- null pointer",
			want:        "null pointer",
		},
		{
			name:        "keeps a file link when the line has other text",
			description: "See [notes.pdf](sp-file://abc) for context",
			want:        "See [notes.pdf](sp-file://abc) for context",
		},
		{
			name:        "skips a list-wrapped image",
			description: "- ![shot](sp-file://abc)\n" + realLine,
			want:        realLine,
		},
		{
			name:        "returns empty when the description is only an image",
			description: "![shot](sp-file://abc)",
			want:        "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, titleFromWorkOrderDescription(tc.description))
		})
	}
}
