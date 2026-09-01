package codex

import (
	"encoding/json"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatCodexJsonLinesEmitsToolRecords(t *testing.T) {
	output := runCodexFormatter(t, []string{
		`{"type":"item.started","item":{"id":"item_1","type":"command_execution","command":"bash -lc git status"}}`,
		`{"type":"item.completed","item":{"id":"item_1","type":"command_execution","command":"bash -lc git status","aggregated_output":"On branch main\n","exit_code":0,"status":"completed"}}`,
		`{"type":"item.completed","item":{"id":"item_2","type":"agent_message","text":"Working tree is clean."}}`,
		`{"type":"item.started","item":{"id":"item_3","type":"file_change","changes":[{"path":"pkg/foo.go","kind":"update"}]}}`,
		`{"type":"item.completed","item":{"id":"item_3","type":"file_change","changes":[{"path":"pkg/foo.go","kind":"update"}]}}`,
	})

	assert.NotContains(t, output, `"type":"item.started"`)
	assert.Contains(t, output, "On branch main")
	assert.Contains(t, output, "Working tree is clean.")
	records := liveLogRecords(t, output)
	require.Len(t, records, 4)
	assert.Equal(t, "tool_start", records[0]["type"])
	assert.Equal(t, "bash", records[0]["kind"])
	assert.Equal(t, "git status", records[0]["text"])
	assert.Equal(t, "tool_end", records[1]["type"])
	assert.Equal(t, "passed", records[1]["status"])
	assert.Equal(t, "edit", records[2]["kind"])
	assert.Equal(t, "pkg/foo.go", records[2]["text"])
}

func TestFormatCodexJsonLinesPairsAnonymousItemIDs(t *testing.T) {
	output := runCodexFormatter(t, []string{
		`{"type":"item.started","item":{"type":"command_execution","command":"true"}}`,
		`{"type":"item.completed","item":{"type":"command_execution","command":"true","aggregated_output":"ok\n","exit_code":0}}`,
	})

	records := liveLogRecords(t, output)
	require.Len(t, records, 2)
	assert.Equal(t, "tool_start", records[0]["type"])
	assert.Equal(t, "bash", records[0]["kind"])
	assert.Equal(t, "tool_end", records[1]["type"])
	assert.Equal(t, "passed", records[1]["status"])
	assert.Equal(t, 1, strings.Count(output, `"type":"tool_start"`))
}

func TestFormatCodexJsonLinesKeepsOverlappingOutputOnTheRightTool(t *testing.T) {
	output := runCodexFormatter(t, []string{
		`{"type":"item.started","item":{"id":"item_a","type":"command_execution","command":"echo a"}}`,
		`{"type":"item.started","item":{"id":"item_b","type":"command_execution","command":"echo b"}}`,
		`{"type":"item.completed","item":{"id":"item_b","type":"command_execution","command":"echo b","aggregated_output":"bbb\n","exit_code":0}}`,
		`{"type":"item.completed","item":{"id":"item_a","type":"command_execution","command":"echo a","aggregated_output":"aaa\n","exit_code":0}}`,
	})

	records := liveLogRecords(t, output)
	require.Len(t, records, 4)
	assert.Equal(t, "item_b", records[0]["id"])
	assert.Equal(t, "tool_end", records[1]["type"])
	assert.Equal(t, "item_b", records[1]["id"])
	assert.Equal(t, "item_a", records[2]["id"])
	assert.Equal(t, "item_a", records[3]["id"])
	assert.Regexp(t, `(?s)"id":"item_b".*bbb.*"type":"tool_end".*"id":"item_a".*aaa`, output)
}

func runCodexFormatter(t *testing.T, lines []string) string {
	t.Helper()
	script, err := filepath.Abs("run.js")
	require.NoError(t, err)
	payload, err := json.Marshal(lines)
	require.NoError(t, err)
	cmd := exec.Command("node", "-e", `const { formatCodexJsonLines } = require(process.argv[1]); formatCodexJsonLines(JSON.parse(process.argv[2]));`, script, string(payload))
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	return string(out)
}

func liveLogRecords(t *testing.T, output string) []map[string]any {
	t.Helper()
	var records []map[string]any
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || !strings.HasPrefix(line, "{") {
			continue
		}
		var rec map[string]any
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			continue
		}
		if rec["type"] == "tool_start" || rec["type"] == "tool_end" {
			records = append(records, rec)
		}
	}
	return records
}
