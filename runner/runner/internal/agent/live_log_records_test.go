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
	writeLiveLogCommandStart(&buf, 1, "echo hello", "", "", startedAt)

	line := liveLogJSONLine(t, buf.String())
	var rec map[string]any
	if err := json.Unmarshal([]byte(line), &rec); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if rec["type"] != "cmd_start" || rec["index"] != float64(1) || rec["text"] != "echo hello" {
		t.Fatalf("unexpected record: %#v", rec)
	}
	if rec["started_at"] != float64(1710000000123) {
		t.Fatalf("expected started_at, got %#v", rec["started_at"])
	}
	if _, hasKind := rec["kind"]; hasKind {
		t.Fatalf("expected kind omitted: %#v", rec)
	}
}

func TestWriteLiveLogCommandStartIncludesKindAndPreview(t *testing.T) {
	var buf bytes.Buffer
	startedAt := time.UnixMilli(1710000000123)
	writeLiveLogCommandStart(&buf, 1, "Set Up Git User", "bash", `echo "hello"`, startedAt)

	line := liveLogJSONLine(t, buf.String())
	var rec map[string]any
	if err := json.Unmarshal([]byte(line), &rec); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if rec["kind"] != "bash" || rec["preview"] != `echo "hello"` {
		t.Fatalf("unexpected kind/preview: %#v", rec)
	}
}

func TestWriteLiveLogCommandEndSeparatesFromIncompleteStdout(t *testing.T) {
	var buf bytes.Buffer
	buf.WriteString("stuck without newline")
	writeLiveLogCommandEnd(&buf, 0, 0, 37*time.Millisecond)

	parts := strings.Split(buf.String(), "\n")
	var jsonLines []string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		jsonLines = append(jsonLines, part)
	}
	if len(jsonLines) != 2 {
		t.Fatalf("want stdout + cmd_end lines, got %#v", jsonLines)
	}
	if jsonLines[0] != "stuck without newline" {
		t.Fatalf("stdout line = %q", jsonLines[0])
	}
	var rec map[string]any
	if err := json.Unmarshal([]byte(jsonLines[1]), &rec); err != nil {
		t.Fatalf("unmarshal cmd_end: %v", err)
	}
	if rec["type"] != "cmd_end" || rec["index"] != float64(0) || rec["status"] != "passed" {
		t.Fatalf("unexpected cmd_end: %#v", rec)
	}
}

func TestDockerCommandsScriptEmitsNewlineBeforeCmdEnd(t *testing.T) {
	script := dockerCommandsScript([]shellDirective{{Text: `printf 'no trailing newline'`, Shell: `printf 'no trailing newline'`}})
	idx := strings.Index(script, `printf '{"type":"cmd_end"`)
	if idx < 0 {
		t.Fatal("missing cmd_end printf")
	}
	before := script[:idx]
	if !strings.HasSuffix(before, "printf '\\n'\n") {
		start := len(before) - 80
		if start < 0 {
			start = 0
		}
		t.Fatalf("expected printf newline before cmd_end; script snippet:\n%s", before[start:])
	}
}

func TestDockerCommandsScriptUsesDisplayNameInCmdStart(t *testing.T) {
	script := dockerCommandsScript([]shellDirective{
		{Text: "Clone", Shell: "git clone repo", Kind: "bash", Preview: "git clone repo"},
	})
	if !strings.Contains(script, `"text":"Clone"`) {
		t.Fatalf("expected named cmd_start text, script:\n%s", script)
	}
	if !strings.Contains(script, `"kind":"bash"`) {
		t.Fatalf("expected kind in cmd_start, script:\n%s", script)
	}
	if !strings.Contains(script, `"preview":"git clone repo"`) {
		t.Fatalf("expected preview in cmd_start, script:\n%s", script)
	}
	if strings.Contains(script, `"text":"git clone repo"`) {
		t.Fatalf("expected display name, not shell, script:\n%s", script)
	}
	if !strings.Contains(script, "git clone repo") {
		t.Fatalf("expected shell body still present, script:\n%s", script)
	}
}

func TestDockerCommandsScriptQuotesPreviewForPrintf(t *testing.T) {
	script := dockerCommandsScript([]shellDirective{
		{Text: "Print", Shell: "true", Kind: "bash", Preview: `printf '%s' foo`},
	})
	if !strings.Contains(script, `printf '%s' `) {
		t.Fatalf("expected literal printf %%s for cmd_start payload, script:\n%s", script)
	}
	if !strings.Contains(script, `'\''`) {
		t.Fatalf("expected apostrophe-safe quoting in preview, script:\n%s", script)
	}
	if strings.Contains(script, `,"started_at":%s}`) {
		t.Fatalf("user preview must not sit in a printf format string, script:\n%s", script)
	}
}

func TestIsLiveLogControlLineRecognizesToolRecords(t *testing.T) {
	if !isLiveLogControlLine(`{"type":"tool_start","kind":"bash","text":"git status","started_at":1}`) {
		t.Fatal("expected tool_start to be a control line")
	}
	if !isLiveLogControlLine(`{"type":"tool_end","kind":"bash","status":"passed","duration_ms":40}`) {
		t.Fatal("expected tool_end to be a control line")
	}
	if isLiveLogControlLine(`{"type":"line","text":"hello"}`) {
		t.Fatal("line must not be a control record")
	}
}

func TestLiveLogPreviewUsesFirstNonEmptyLine(t *testing.T) {
	if got := liveLogPreview("\n  echo hello\nworld"); got != "echo hello" {
		t.Fatalf("preview=%q", got)
	}
	long := strings.Repeat("a", 100)
	if got := liveLogPreview(long); len([]rune(got)) != 80 {
		t.Fatalf("truncated len=%d", len([]rune(got)))
	}
}

func liveLogJSONLine(t *testing.T, raw string) string {
	t.Helper()
	for _, part := range strings.Split(raw, "\n") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		return part
	}
	t.Fatal("no JSON line in live log write")
	return ""
}
