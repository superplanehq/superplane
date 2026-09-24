package livelogs

import "testing"

func TestParseRunnerControlRecord(t *testing.T) {
	t.Run("cmd_start", func(t *testing.T) {
		rec, ok := parseRunnerControlRecord(`{"type":"cmd_start","index":2,"text":"echo hello","started_at":1710000000123}`)
		if !ok {
			t.Fatalf("expected cmd_start to parse")
		}
		if rec["type"] != "cmd_start" || rec["index"] != 2 || rec["text"] != "echo hello" || rec["started_at"] != int64(1710000000123) {
			t.Fatalf("unexpected cmd_start record: %#v", rec)
		}
	})

	t.Run("cmd_start with kind and preview", func(t *testing.T) {
		rec, ok := parseRunnerControlRecord(`{"type":"cmd_start","index":2,"text":"Set Up Git User","kind":"bash","preview":"echo hi","started_at":1}`)
		if !ok {
			t.Fatalf("expected cmd_start to parse")
		}
		if rec["kind"] != "bash" || rec["preview"] != "echo hi" {
			t.Fatalf("unexpected kind/preview: %#v", rec)
		}
	})

	t.Run("tool_start", func(t *testing.T) {
		rec, ok := parseRunnerControlRecord(`{"type":"tool_start","id":"toolu_a","kind":"read","text":"pkg/foo.go","started_at":9}`)
		if !ok {
			t.Fatalf("expected tool_start to parse")
		}
		if rec["type"] != "tool_start" || rec["id"] != "toolu_a" || rec["kind"] != "read" || rec["text"] != "pkg/foo.go" || rec["started_at"] != int64(9) {
			t.Fatalf("unexpected tool_start: %#v", rec)
		}
	})

	t.Run("tool_end", func(t *testing.T) {
		rec, ok := parseRunnerControlRecord(`{"type":"tool_end","id":"toolu_a","kind":"bash","status":"passed","duration_ms":40}`)
		if !ok {
			t.Fatalf("expected tool_end to parse")
		}
		if rec["type"] != "tool_end" || rec["id"] != "toolu_a" || rec["kind"] != "bash" || rec["status"] != "passed" || rec["duration_ms"] != int64(40) {
			t.Fatalf("unexpected tool_end: %#v", rec)
		}
	})

	t.Run("cmd_start without started_at", func(t *testing.T) {
		rec, ok := parseRunnerControlRecord(`{"type":"cmd_start","index":2,"text":"echo hello"}`)
		if !ok {
			t.Fatalf("expected cmd_start to parse")
		}
		if _, hasStartedAt := rec["started_at"]; hasStartedAt {
			t.Fatalf("expected started_at to be omitted: %#v", rec)
		}
	})

	t.Run("cmd_end", func(t *testing.T) {
		rec, ok := parseRunnerControlRecord(`{"type":"cmd_end","index":2,"status":"passed","duration_ms":37}`)
		if !ok {
			t.Fatalf("expected cmd_end to parse")
		}
		if rec["type"] != "cmd_end" || rec["index"] != 2 || rec["status"] != "passed" || rec["duration_ms"] != int64(37) {
			t.Fatalf("unexpected cmd_end record: %#v", rec)
		}
	})

	t.Run("invalid", func(t *testing.T) {
		cases := []string{
			`{"type":"line","text":"hello"}`,
			`{"type":"cmd_start","index":-1,"text":"echo hello"}`,
			`{"type":"cmd_end","index":0,"status":"running","duration_ms":10}`,
			`{"type":"tool_end","status":"running","duration_ms":10}`,
			`not-json`,
		}
		for _, item := range cases {
			if _, ok := parseRunnerControlRecord(item); ok {
				t.Fatalf("expected invalid record to fail: %s", item)
			}
		}
	})
}
