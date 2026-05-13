package agents

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
)

func TestDeriveChatTitle_TruncatesMultiByteSafely(t *testing.T) {
	// Each "你" is 3 bytes in UTF-8. A byte-level slice at 59 would land
	// inside the rune at position 19 and produce invalid UTF-8.
	long := strings.Repeat("你", chatTitleMaxLength+10)
	got := deriveChatTitle(long)
	assert.True(t, utf8.ValidString(got), "title must be valid UTF-8 even when truncated mid-content")
	assert.True(t, strings.HasSuffix(got, "…"))
}

func TestDeriveChatTitle_TakesFirstNonEmptyLine(t *testing.T) {
	got := deriveChatTitle("\n\nhello world\nsecond line")
	assert.Equal(t, "hello world", got)
}

func TestDeriveChatTitle_EmptyInput(t *testing.T) {
	assert.Equal(t, "", deriveChatTitle(""))
	assert.Equal(t, "", deriveChatTitle("   \n\n  "))
}
