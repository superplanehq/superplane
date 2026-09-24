package jira

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// adfBlockSeparator is the blank line between two blocks of a converted
// document, so paragraphs and lists stay apart in the plain text.
const adfBlockSeparator = "\n\n"

// ADFToText renders a Jira rich text value as readable plain text.
//
// Jira Cloud answers with a rich text field such as an issue description
// either as an Atlassian Document Format document or, on a site that still
// sends the legacy representation, as a wiki markup string. Both shapes reach
// SuperPlane through webhooks and through the REST API, and a caller that
// interpolates a document into a string writes Go map formatting into
// user-facing text. A value that carries no readable text becomes an empty
// string.
func ADFToText(value any) string {
	return strings.TrimSpace(adfValueText(normalizeADFValue(value)))
}

func adfValueText(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case []any:
		return adfNodesText(typed, adfBlockSeparator)
	case map[string]any:
		return adfNodeText(typed)
	default:
		return ""
	}
}

// normalizeADFValue reduces the shapes a document arrives in - a decoded
// webhook body, an *ADFDoc built in Go - to the generic JSON types the
// conversion reads. A value that does not survive the round trip is dropped.
func normalizeADFValue(value any) any {
	switch value.(type) {
	case nil, string, map[string]any, []any:
		return value
	}

	encoded, err := json.Marshal(value)
	if err != nil {
		return nil
	}

	var normalized any
	if err := json.Unmarshal(encoded, &normalized); err != nil {
		return nil
	}

	return normalized
}

func adfNodeText(node map[string]any) string {
	switch adfNodeType(node) {
	case "text":
		return adfLinkedText(node)
	case "hardBreak":
		return "\n"
	case "mention", "emoji":
		return adfAttribute(node, "text")
	case "inlineCard", "blockCard", "embedCard":
		return adfAttribute(node, "url")
	case "rule":
		return "---"
	case "paragraph", "tableCell", "tableHeader":
		return adfNodesText(adfContent(node), "")
	case "heading":
		return strings.Repeat("#", adfHeadingLevel(node)) + " " + adfNodesText(adfContent(node), "")
	case "codeBlock":
		return "```\n" + adfNodesText(adfContent(node), "") + "\n```"
	case "bulletList", "orderedList":
		return adfListText(node)
	default:
		return adfNodesText(adfContent(node), adfBlockSeparator)
	}
}

func adfNodesText(nodes []any, separator string) string {
	parts := make([]string, 0, len(nodes))
	for _, node := range nodes {
		if text := adfValueText(node); text != "" {
			parts = append(parts, text)
		}
	}

	return strings.Join(parts, separator)
}

// adfListText writes each item on its own line and keeps a nested list under
// the marker of the item that holds it.
func adfListText(node map[string]any) string {
	ordered := adfNodeType(node) == "orderedList"

	lines := make([]string, 0, len(adfContent(node)))
	for _, item := range adfContent(node) {
		text := strings.TrimSpace(adfValueText(item))
		if text == "" {
			continue
		}

		marker := "- "
		if ordered {
			marker = strconv.Itoa(len(lines)+1) + ". "
		}
		lines = append(lines, marker+strings.ReplaceAll(text, "\n", "\n"+strings.Repeat(" ", len(marker))))
	}

	return strings.Join(lines, "\n")
}

// adfLinkedText keeps the target of a link next to its label, so a reader of
// the converted text can still follow it.
func adfLinkedText(node map[string]any) string {
	text, _ := node["text"].(string)
	href := adfLinkHref(node)
	if text == "" || href == "" || href == text {
		return text
	}

	return fmt.Sprintf("[%s](%s)", text, href)
}

func adfLinkHref(node map[string]any) string {
	marks, _ := node["marks"].([]any)
	for _, entry := range marks {
		mark, ok := entry.(map[string]any)
		if ok && adfNodeType(mark) == "link" {
			return adfAttribute(mark, "href")
		}
	}

	return ""
}

func adfNodeType(node map[string]any) string {
	nodeType, _ := node["type"].(string)
	return nodeType
}

func adfContent(node map[string]any) []any {
	content, _ := node["content"].([]any)
	return content
}

func adfAttribute(node map[string]any, name string) string {
	attributes, _ := node["attrs"].(map[string]any)
	value, _ := attributes[name].(string)
	return value
}

func adfHeadingLevel(node map[string]any) int {
	attributes, _ := node["attrs"].(map[string]any)

	level := 1
	switch value := attributes["level"].(type) {
	case float64:
		level = int(value)
	case int:
		level = value
	}
	if level < 1 || level > 6 {
		return 1
	}

	return level
}
