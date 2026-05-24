package agent

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestWriteLiveLogCommandStartIncludesStartedAt(t *testing.T) {
	var buf bytes.Buffer
	startedAt := time.UnixMilli(1710000000123)
	writeLiveLogCommandStart(&buf, 1, "echo hello", startedAt)

	var rec map[string]any
	if err := json.Unmarshal(buf.Bytes(), &rec); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if rec["type"] != "cmd_start" || rec["index"] != float64(1) || rec["text"] != "echo hello" {
		t.Fatalf("unexpected record: %#v", rec)
	}
	if rec["started_at"] != float64(1710000000123) {
		t.Fatalf("expected started_at, got %#v", rec["started_at"])
	}
	if !strings.HasSuffix(buf.String(), "\n") {
		t.Fatalf("expected trailing newline")
	}
}
