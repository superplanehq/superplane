package elastic

import (
	"fmt"
	"slices"
	"strings"
)

func isExpressionValue(value string) bool {
	return strings.Contains(value, "{{")
}

func ensureIndexExists(client *Client, index string) error {
	if isExpressionValue(index) {
		return nil
	}

	indices, err := client.ListIndices()
	if err != nil {
		return fmt.Errorf("failed to list indices: %w", err)
	}

	if slices.ContainsFunc(indices, func(item IndexInfo) bool {
		return item.Index == index
	}) {
		return nil
	}

	return fmt.Errorf("index %q was not found", index)
}

func ensureDocumentExists(client *Client, index, document string) error {
	if isExpressionValue(index) || isExpressionValue(document) {
		return nil
	}

	resp, err := client.GetDocument(index, document)
	if err != nil {
		return fmt.Errorf("failed to verify document %q in index %q: %w", document, index, err)
	}
	if !resp.Found {
		return fmt.Errorf("document %q was not found in index %q", document, index)
	}

	return nil
}
