package jira

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test__ADFToText(t *testing.T) {
	t.Run("nil and unreadable values report no text", func(t *testing.T) {
		assert.Equal(t, "", ADFToText(nil))
		assert.Equal(t, "", ADFToText(42))
		assert.Equal(t, "", ADFToText(map[string]any{"type": "doc"}))
	})

	t.Run("a legacy wiki markup string passes through", func(t *testing.T) {
		assert.Equal(t, "Refunds retry twice.", ADFToText("  Refunds retry twice.  "))
	})

	t.Run("a document built in Go reads back as its text", func(t *testing.T) {
		assert.Equal(t, "Refunds retry twice.", ADFToText(WrapInADF("Refunds retry twice.")))
	})

	t.Run("paragraphs are separated by a blank line", func(t *testing.T) {
		document := decodeADF(t, `{
			"type": "doc",
			"version": 1,
			"content": [
				{"type": "paragraph", "content": [{"type": "text", "text": "A retry charges twice."}]},
				{"type": "paragraph", "content": [{"type": "text", "text": "Cap the retries at 3."}]}
			]
		}`)

		assert.Equal(t, "A retry charges twice.\n\nCap the retries at 3.", ADFToText(document))
	})

	t.Run("headings, lists, code, and rules keep their structure", func(t *testing.T) {
		document := decodeADF(t, `{
			"type": "doc",
			"version": 1,
			"content": [
				{"type": "heading", "attrs": {"level": 2}, "content": [{"type": "text", "text": "Steps"}]},
				{"type": "bulletList", "content": [
					{"type": "listItem", "content": [
						{"type": "paragraph", "content": [{"type": "text", "text": "Open the refund page."}]}
					]},
					{"type": "listItem", "content": [
						{"type": "paragraph", "content": [{"type": "text", "text": "Retry the payment."}]}
					]}
				]},
				{"type": "rule"},
				{"type": "codeBlock", "content": [{"type": "text", "text": "refund --retry"}]}
			]
		}`)

		assert.Equal(
			t,
			"## Steps\n\n- Open the refund page.\n- Retry the payment.\n\n---\n\n```\nrefund --retry\n```",
			ADFToText(document),
		)
	})

	t.Run("an ordered list numbers its items", func(t *testing.T) {
		document := decodeADF(t, `{
			"type": "doc",
			"content": [
				{"type": "orderedList", "content": [
					{"type": "listItem", "content": [
						{"type": "paragraph", "content": [{"type": "text", "text": "First"}]}
					]},
					{"type": "listItem", "content": [
						{"type": "paragraph", "content": [{"type": "text", "text": "Second"}]}
					]}
				]}
			]
		}`)

		assert.Equal(t, "1. First\n2. Second", ADFToText(document))
	})

	t.Run("a link keeps its target and a hard break its line", func(t *testing.T) {
		document := decodeADF(t, `{
			"type": "doc",
			"content": [
				{"type": "paragraph", "content": [
					{"type": "text", "text": "See "},
					{
						"type": "text",
						"text": "the runbook",
						"marks": [{"type": "link", "attrs": {"href": "https://example.com/runbook"}}]
					},
					{"type": "hardBreak"},
					{"type": "text", "text": "Then retry."}
				]}
			]
		}`)

		assert.Equal(
			t,
			"See [the runbook](https://example.com/runbook)\nThen retry.",
			ADFToText(document),
		)
	})

	t.Run("mentions and inline cards read as text", func(t *testing.T) {
		document := decodeADF(t, `{
			"type": "doc",
			"content": [
				{"type": "paragraph", "content": [
					{"type": "mention", "attrs": {"id": "acct-1", "text": "@Ana"}},
					{"type": "text", "text": " owns "},
					{"type": "inlineCard", "attrs": {"url": "https://example.com/ENG-1"}}
				]}
			]
		}`)

		assert.Equal(t, "@Ana owns https://example.com/ENG-1", ADFToText(document))
	})
}

func Test__IssueDescriptionText(t *testing.T) {
	assert.Equal(t, "", IssueDescriptionText(nil))
	assert.Equal(t, "", IssueDescriptionText(&Issue{Key: "ENG-1"}))

	issue := &Issue{
		Key:    "ENG-1",
		Fields: map[string]any{"description": decodeADF(t, `{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"Retry loops."}]}]}`)},
	}
	assert.Equal(t, "Retry loops.", IssueDescriptionText(issue))
}

func decodeADF(t *testing.T, document string) map[string]any {
	t.Helper()

	decoded := map[string]any{}
	require.NoError(t, json.Unmarshal([]byte(document), &decoded))
	return decoded
}
