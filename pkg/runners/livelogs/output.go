// Package livelogs streams runner task output as NDJSON for the live-log API.
package livelogs

import (
	"io"
	"net/http"
	"strings"
)

// StreamTextOutputToNDJSON writes stored task output as line records (local dev without CloudWatch).
func StreamTextOutputToNDJSON(w io.Writer, flusher http.Flusher, output string) error {
	nw := ndjsonWriter{w: w, f: flusher}
	text := strings.TrimSpace(output)
	if text == "" {
		return nil
	}
	for _, line := range strings.Split(text, "\n") {
		if err := nw.writeRecord(map[string]any{"type": "line", "text": line}); err != nil {
			return err
		}
	}
	return nil
}
