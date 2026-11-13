package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/runners/api"
	"github.com/superplanehq/superplane/pkg/runners/logspool"
	"github.com/superplanehq/superplane/pkg/runners/models"
)

func TestAgentPreservesFailureDiagnostics(t *testing.T) {
	for _, pipe := range []string{"0", "1"} {
		t.Run("pipe="+pipe, func(t *testing.T) {
			if pipe == "0" {
				skipPTYIntegrationOnCI(t)
			}
			t.Setenv("RUNNER_SHELL_USE_PIPE", pipe)
			for _, tc := range []struct {
				name, command, cause string
				code                 int
			}{
				{"prompt", `printf '%s\n' '{"type":"result","is_error":true,"error":"Provider unavailable"}' > "$SUPERPLANE_RESULT_FILE"; echo 'provider stderr' >&2; return 42`, "Provider unavailable", 42},
				{"shell", `printf 'Authentication failed: %s Bearer unknown-credential\n' "$PRIVATE_VALUE" >&2; return 23`, "Authentication failed", 23},
				{"oversized result", `head -c 20000 /dev/zero | tr '\0' x > "$SUPERPLANE_RESULT_FILE"; echo 'original error' >&2; return 24`, "original error", 24},
				{"silent", `return 7`, "No diagnostic output", 7},
				{"bounded output", `for i in {1..2000}; do echo 'diagnostic output before the final error'; done; printf 'final error: %s\n' "$PRIVATE_VALUE" >&2; return 9`, "final error", 9},
				{"Claude result", `printf '%s\n' '{"type":"result","is_error":true,"result":"Claude provider unavailable"}' > "$SUPERPLANE_RESULT_FILE"; return 10`, "Claude provider unavailable", 10},
				{"ordinary settings", `echo 'HTTP 401 after 3600 seconds: certificate rejected' >&2; return 1`, "HTTP 401 after 3600 seconds: certificate rejected", 1},
				{"JSON environment secret", `printf '%s\n' '{"message":"pa\"ss&word"}' >&2; return 23`, "message", 23},
				{"JSON query credential", `printf '%s\n' '{"message":"https://example.com/?status=401\u0026key=private-query-value"}' >&2; return 23`, "status=401", 23},
				{"quoted credential", `printf '%s\n' '{"password":"first second&third\"fourth"}' >&2; return 23`, "[REDACTED]", 23},
				{"prompt timeout", `printf '%s\n' '{"is_error":true,"error":"old provider error"}' > "$SUPERPLANE_RESULT_FILE"; exec sleep 30`, "execution timed out", 124},
			} {
				t.Run(tc.name, func(t *testing.T) {
					server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						_, _ = io.Copy(io.Discard, r.Body)
						w.WriteHeader(http.StatusNoContent)
					}))
					defer server.Close()
					spool, err := logspool.New(logspool.Config{Directory: t.TempDir(), TaskID: "diagnostic-task", BaseURL: server.URL, AccessToken: "test-access-token", ChunkBytes: 65536, MaxBytes: 1048576})
					require.NoError(t, err)
					defer func() { require.NoError(t, spool.Close()); require.NoError(t, spool.Wait(t.Context())) }()
					runner := &Agent{Config: Config{MaxOutputBytes: 8192, MaxExecutionSeconds: 30, TaskWorkDir: t.TempDir()}}
					stepName, kind := "Check pull requests", "bash"
					if tc.name == "prompt" || tc.name == "Claude result" || tc.name == "prompt timeout" {
						stepName, kind = "Implementation", "prompt"
					}
					if tc.name == "ordinary settings" {
						stepName = "Check pull requests 1"
					}
					task := &api.TaskPayload{ID: "diagnostic-task", ExecutionMode: "host", Environment: []api.EnvironmentVariable{
						{Name: "PRIVATE_VALUE", Value: "private-environment-value"},
						{Name: "SERIALIZED_SECRET", Value: `pa"ss&word`},
						{Name: "CHECKOUT_COMMIT_SETUP", Value: "1"},
						{Name: "SUPERPLANE_EXECUTION_TIMEOUT_SECONDS", Value: "3600"},
						{Name: "PAGER", Value: "cat"},
					}, Commands: models.CommandList{
						{Name: "Earlier prompt", Kind: "prompt", Command: `printf '%s\n' '{"type":"result","is_error":false,"error":"stale provider error"}' > "$SUPERPLANE_RESULT_FILE"; head -c 30000 /dev/zero | tr '\0' x; echo`},
						{Name: stepName, Kind: kind, Command: tc.command},
						{Name: "Later check", Command: `echo 'must not run'`},
					}}
					ctx := t.Context()
					if tc.name == "prompt timeout" {
						var cancel context.CancelFunc
						ctx, cancel = context.WithTimeout(ctx, 2*time.Second)
						defer cancel()
					}
					result := runner.execute(ctx, task, spool, make(chan string))
					assert.Equal(t, tc.code, result.ExitCode)
					message := result.errorMessage()
					assert.Contains(t, message, stepName)
					assert.NotContains(t, message, "stale provider error")
					assert.Contains(t, message, fmt.Sprintf("exit code %d", tc.code))
					assert.Contains(t, message, tc.cause)
					assert.Contains(t, message, "diagnostic-task")
					assert.NotContains(t, message, "private-environment-value")
					assert.NotContains(t, message, "unknown-credential")
					for _, secret := range []string{`pa\"ss&word`, "private-query-value", "second&", "third", "fourth", "old provider error"} {
						assert.NotContains(t, message, secret)
					}
					assert.NotContains(t, message, "must not run")
					assert.LessOrEqual(t, len(message), 4096)
					if tc.name == "prompt" {
						assert.True(t, json.Valid(result.Result))
					}
				})
			}
		})
	}
}

func TestFailureDiagnosticsRetainsLateOutputForFailedStep(t *testing.T) {
	for _, nextStep := range []bool{false, true} {
		t.Run(fmt.Sprintf("next step=%t", nextStep), func(t *testing.T) {
			diagnostics := &failureDiagnostics{}
			_, err := io.WriteString(diagnostics, `{"type":"cmd_start","index":0,"text":"Check pull requests"}`+"\n"+
				`{"type":"cmd_end","index":0,"status":"failed"}`+"\n"+"late failure output\n")
			require.NoError(t, err)
			if nextStep {
				_, err = io.WriteString(diagnostics, `{"type":"cmd_start","index":1,"text":"Cleanup"}`+"\n"+"later cleanup output\n")
				require.NoError(t, err)
			}
			failure := diagnostics.failure(&api.TaskPayload{ID: "task-1"}, 23, errors.New("exit code 23"), nil)
			assert.Contains(t, failure.Error(), "Check pull requests")
			assert.Contains(t, failure.Error(), "late failure output")
			assert.NotContains(t, failure.Error(), "later cleanup output")
		})
	}
}

func TestPromptFailureCauseWithDockerExitError(t *testing.T) {
	for _, cause := range []string{"exit status 42", "runner transport unavailable"} {
		t.Run(cause, func(t *testing.T) {
			diagnostics := &failureDiagnostics{step: liveLogCommandStartRecord{Kind: "prompt", Text: "Implementation"}}
			failure := diagnostics.failure(&api.TaskPayload{ID: "task-1"}, 42, errors.New(cause), json.RawMessage(`{"is_error":true,"error":"Provider unavailable"}`))
			if cause == "exit status 42" {
				assert.Contains(t, failure.Error(), "Provider unavailable")
				return
			}
			assert.Contains(t, failure.Error(), cause)
			assert.NotContains(t, failure.Error(), "Provider unavailable")
		})
	}
}

func BenchmarkFailureDiagnosticsOutput(b *testing.B) {
	chunk := bytes.Repeat([]byte("diagnostic output\n"), 4096)
	diagnostics := &failureDiagnostics{}
	b.SetBytes(int64(len(chunk)))
	b.ReportAllocs()
	for b.Loop() {
		_, _ = diagnostics.Write(chunk)
	}
}
