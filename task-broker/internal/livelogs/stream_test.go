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
			`not-json`,
		}
		for _, item := range cases {
			if _, ok := parseRunnerControlRecord(item); ok {
				t.Fatalf("expected invalid record to fail: %s", item)
			}
		}
	})
}
