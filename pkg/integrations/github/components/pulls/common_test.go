package pulls

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test__AuthorLogin(t *testing.T) {
	t.Run("reads the comment author login", func(t *testing.T) {
		data := map[string]any{
			"comment": map[string]any{"user": map[string]any{"login": "coderabbitai[bot]"}},
		}
		assert.Equal(t, "coderabbitai[bot]", authorLogin("issue_comment", data))
	})

	t.Run("reads the review author login", func(t *testing.T) {
		data := map[string]any{
			"review": map[string]any{"user": map[string]any{"login": "jules"}},
		}
		assert.Equal(t, "jules", authorLogin("pull_request_review", data))
	})

	t.Run("returns empty string when there is no author", func(t *testing.T) {
		assert.Equal(t, "", authorLogin("issue_comment", map[string]any{}))
	})
}

func Test__IsAllowedBot(t *testing.T) {
	botData := func(login string) map[string]any {
		return map[string]any{
			"comment": map[string]any{"user": map[string]any{"login": login, "type": "Bot"}},
		}
	}
	userData := func(login string) map[string]any {
		return map[string]any{
			"comment": map[string]any{"user": map[string]any{"login": login, "type": "User"}},
		}
	}

	t.Run("empty allowlist never matches", func(t *testing.T) {
		assert.False(t, isAllowedBot("issue_comment", botData("coderabbitai"), nil))
	})

	t.Run("bot login on the allowlist matches", func(t *testing.T) {
		assert.True(t, isAllowedBot("issue_comment", botData("coderabbitai"), []string{"coderabbitai"}))
	})

	t.Run("bot login with a [bot] suffix matches a bare allowlist entry", func(t *testing.T) {
		assert.True(t, isAllowedBot("issue_comment", botData("coderabbitai[bot]"), []string{"coderabbitai"}))
	})

	t.Run("allowlist entry with a [bot] suffix matches a bare bot login", func(t *testing.T) {
		assert.True(t, isAllowedBot("issue_comment", botData("coderabbitai"), []string{"coderabbitai[bot]"}))
	})

	t.Run("match is case-insensitive", func(t *testing.T) {
		assert.True(t, isAllowedBot("issue_comment", botData("CodeRabbitAI"), []string{"coderabbitai"}))
	})

	t.Run("allowlist entry with a leading @ still matches", func(t *testing.T) {
		assert.True(t, isAllowedBot("issue_comment", botData("coderabbitai"), []string{"@coderabbitai"}))
	})

	t.Run("bot login not on the allowlist does not match", func(t *testing.T) {
		assert.False(t, isAllowedBot("issue_comment", botData("codecov"), []string{"coderabbitai"}))
	})

	t.Run("non-bot author matching an allowlist name does not match", func(t *testing.T) {
		assert.False(t, isAllowedBot("issue_comment", userData("coderabbitai"), []string{"coderabbitai"}))
	})
}
