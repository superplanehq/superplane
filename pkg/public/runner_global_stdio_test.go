package public

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/png"
	"io"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/blob"
	"github.com/superplanehq/superplane/pkg/blob/filesystem"
	runner "github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func TestGlobalRunnerToolsOverStdio(t *testing.T) {
	r := support.Setup(t)
	t.Setenv("JWT_SECRET", "test")
	t.Setenv("BASE_URL", "http://files.test")
	store, err := filesystem.New(t.TempDir())
	require.NoError(t, err)
	blob.SetCurrent(store)
	t.Cleanup(func() { blob.SetCurrent(nil) })

	db := database.DB(t.Context())
	server := newRunnerArtifactTestServer(t, r, jwt.NewSigner("test"))
	ctx, factory, order := mustTaskLinkedAgentRun(t, r)
	member := mustTaskOwnerMember(t, r, "contract-dev")
	env := runner.AttachRunnerEnv(ctx, nil, 60, nil)
	mcp := startGlobalMCP(t, server, runnerTokenFrom(env), runner.AppendGlobalMCP(env, nil))

	assert.ElementsMatch(t, everyGlobalRunnerTool, mcp.toolNames())

	members := mcp.call("list_members", map[string]any{})["members"].([]any)
	assert.Contains(t, fieldValues(members, "user_id"), member.ID.String())

	assigned := mcp.call("assign_task_owner", map[string]any{"user": member.ID.String(), "reason": "Owns the change."})
	assert.Equal(t, "assigned", assigned["status"])
	assignees, err := order.ListAssignees(db)
	require.NoError(t, err)
	require.Len(t, assignees, 1)
	assert.Equal(t, member.ID, assignees[0].UserID)

	suggestion := mcp.call("suggest_task_owner", map[string]any{})
	assert.Contains(t, fieldValues(suggestion["members"].([]any), "user_id"), member.ID.String())

	created := mcp.call("create_task", map[string]any{"title": "Follow up on retries"})
	assert.Equal(t, "Follow up on retries", created["title"])
	tasks := mcp.call("list_tasks", map[string]any{})["tasks"].([]any)
	assert.Contains(t, fieldValues(tasks, "key"), created["key"])

	added := mcp.call("add_checks", map[string]any{"checks": []map[string]any{{
		"key": "api-latency", "name": "API latency", "score": 2, "max_score": 5, "level": "caution", "reason": "p99 grows by 40 ms.",
	}}})
	assert.Equal(t, []string{"api-latency"}, fieldValues(added["checks"].([]any), "key"))
	checks, err := order.ListChecks(db)
	require.NoError(t, err)
	require.Len(t, checks, 1)
	assert.Equal(t, models.FactoryWorkOrderCheckSourceAgent, checks[0].Source)
	listed := mcp.call("list_checks", map[string]any{"task": factory.WorkOrderKey(order.Number)})
	assert.Equal(t, []string{"api-latency"}, fieldValues(listed["checks"].([]any), "key"))
	reserved := mcp.callError("add_checks", map[string]any{"checks": []map[string]any{{
		"key": models.PlanningClarityCheckKey, "name": "Clarity", "score": 3, "max_score": 3, "level": "positive", "reason": "Clear.",
	}}})
	assert.Contains(t, reserved, "belongs to SuperPlane")
	removed := mcp.call("remove_checks", map[string]any{"keys": []string{"api-latency"}})
	assert.Equal(t, []any{"api-latency"}, removed["removed"])

	screenshot := filepath.Join(mcp.taskDir, "evidence", "checkout.png")
	require.NoError(t, os.MkdirAll(filepath.Dir(screenshot), 0700))
	require.NoError(t, os.WriteFile(screenshot, onePixelPNG(t), 0600))
	inspected := mcp.callTool("get_artifact", map[string]any{"path": "evidence/checkout.png"})
	require.NotEqual(t, true, inspected["isError"], inspected)
	assert.Contains(t, fieldValues(inspected["content"].([]any), "type"), "image")
	mcp.call("add_artifact", map[string]any{"type": "file", "path": "evidence/checkout.png", "title": "Checkout", "key": "checkout"})
	mcp.call("add_artifact", map[string]any{"type": "link", "url": "https://preview.example.com", "title": "Preview", "key": "preview"})
	mcp.call("add_artifact", map[string]any{"type": "markdown", "body": "Retries stop after 3 attempts.", "title": "Notes", "key": "notes"})
	artifacts := mcp.call("list_artifacts", map[string]any{})["artifacts"].([]any)
	assert.ElementsMatch(t, []string{"checkout", "preview", "notes"}, fieldValues(artifacts, "key"))
	note := mcp.call("get_artifact", map[string]any{"key": "notes"})
	assert.Equal(t, "Retries stop after 3 attempts.", note["body"])
	mcp.call("remove_artifact", map[string]any{"key": "preview"})
	_, err = findRunnerArtifactByKey(db, order, "preview")
	assert.ErrorIs(t, err, models.ErrFactoryWorkOrderArtifactNotFound)
	stored, err := order.ListArtifacts(db)
	require.NoError(t, err)
	assert.Len(t, stored, 2)

	closed := mcp.call("close_task", map[string]any{"result": "rejected"})
	assert.Equal(t, models.FactoryWorkOrderStateClosed, closed["state"])
	reloaded, err := factory.FindWorkOrder(db, order.ID)
	require.NoError(t, err)
	assert.Equal(t, models.FactoryWorkOrderStateClosed, reloaded.State)
}

func fieldValues(items []any, field string) []string {
	values := []string{}
	for _, item := range items {
		value, _ := item.(map[string]any)[field].(string)
		values = append(values, value)
	}
	return values
}

func onePixelPNG(t *testing.T) []byte {
	t.Helper()
	var buffer bytes.Buffer
	require.NoError(t, png.Encode(&buffer, image.NewRGBA(image.Rect(0, 0, 1, 1))))
	return buffer.Bytes()
}

// globalMCPSession runs the global runner MCP server from the task files of
// a run, as a runner does, against the real API handlers.
type globalMCPSession struct {
	t       *testing.T
	taskDir string
	stdin   io.Writer
	stdout  *bufio.Scanner
	stderr  *bytes.Buffer
	nextID  int
}

func startGlobalMCP(t *testing.T, server *Server, token string, files []runner.TaskFile) *globalMCPSession {
	t.Helper()
	api := httptest.NewServer(server.Router)
	t.Cleanup(api.Close)
	taskDir := t.TempDir()
	for _, file := range files {
		path := filepath.Join(taskDir, file.Path)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0700))
		require.NoError(t, os.WriteFile(path, []byte(file.Content), 0600))
	}

	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	t.Cleanup(cancel)
	command := exec.CommandContext(ctx, "node", filepath.Join(taskDir, "superplane_mcp/server.js"))
	command.Env = append(os.Environ(), "SUPERPLANE_BASE_URL="+api.URL, "SUPERPLANE_RUNNER_TOKEN="+token, "SUPERPLANE_TASK_DIR="+taskDir)
	stdin, err := command.StdinPipe()
	require.NoError(t, err)
	stdout, err := command.StdoutPipe()
	require.NoError(t, err)
	stderr := &bytes.Buffer{}
	command.Stderr = stderr
	require.NoError(t, command.Start())
	t.Cleanup(func() { _ = command.Process.Kill(); _ = command.Wait() })

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	session := &globalMCPSession{t: t, taskDir: taskDir, stdin: stdin, stdout: scanner, stderr: stderr}
	session.request("initialize", map[string]any{})
	return session
}

func (s *globalMCPSession) request(method string, params any) map[string]any {
	s.t.Helper()
	s.nextID++
	require.NoError(s.t, json.NewEncoder(s.stdin).Encode(map[string]any{"jsonrpc": "2.0", "id": s.nextID, "method": method, "params": params}))
	require.True(s.t, s.stdout.Scan(), s.stderr.String())
	var response map[string]any
	require.NoError(s.t, json.Unmarshal(s.stdout.Bytes(), &response))
	require.Nil(s.t, response["error"], method)
	return response["result"].(map[string]any)
}

func (s *globalMCPSession) toolNames() []string {
	s.t.Helper()
	names := []string{}
	for _, tool := range s.request("tools/list", map[string]any{})["tools"].([]any) {
		names = append(names, tool.(map[string]any)["name"].(string))
	}
	return names
}

func (s *globalMCPSession) callTool(name string, arguments map[string]any) map[string]any {
	s.t.Helper()
	return s.request("tools/call", map[string]any{"name": name, "arguments": arguments})
}

// call runs a tool that must succeed and returns its structured result.
func (s *globalMCPSession) call(name string, arguments map[string]any) map[string]any {
	s.t.Helper()
	result := s.callTool(name, arguments)
	require.NotEqual(s.t, true, result["isError"], "%s: %v", name, result)
	structured, _ := result["structuredContent"].(map[string]any)
	return structured
}

// callError runs a tool that must fail and returns its error text.
func (s *globalMCPSession) callError(name string, arguments map[string]any) string {
	s.t.Helper()
	result := s.callTool(name, arguments)
	require.Equal(s.t, true, result["isError"], "%s: %v", name, result)
	return result["content"].([]any)[0].(map[string]any)["text"].(string)
}
