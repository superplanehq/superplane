package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/extensions/hub/protocol"
)

func TestRuntimeExecutorDownloadsOnceAndReusesCache(t *testing.T) {
	t.Parallel()

	transport := &fakeRoundTripper{
		statusCode: http.StatusOK,
		body:       "export async function invoke(payload) { return payload; }",
	}
	runner := &fakeRunner{
		output: `{"ok":true}`,
	}

	executor := NewRuntimeExecutor(RuntimeExecutorConfig{
		HubURL:     "http://hub.example",
		CacheDir:   t.TempDir(),
		DenoBinary: "deno",
		HTTPClient: &http.Client{Transport: transport},
		Runner:     runner,
	})

	message := protocol.JobAssignMessage{
		JobID:       "job-1",
		ExtensionID: "ext-1",
		VersionID:   "ver-1",
		Digest:      "sha256:test",
		BundleToken: "bundle-token",
		Invocation:  json.RawMessage(`{"input":{"value":"hello"}}`),
	}

	output, err := executor.HandleJob(context.Background(), message)
	require.NoError(t, err)
	require.JSONEq(t, `{"ok":true}`, string(output))
	require.Len(t, transport.requests, 1)
	require.Contains(t, transport.requests[0], "/api/v1/extensions/ext-1/versions/ver-1/bundle.js")
	require.Contains(t, transport.requests[0], "token=bundle-token")
	_, err = os.Stat(filepath.Join(executor.cacheDir, "ext-1", "ver-1", "sha256:test", "bundle.js"))
	require.NoError(t, err)

	output, err = executor.HandleJob(context.Background(), message)
	require.NoError(t, err)
	require.JSONEq(t, `{"ok":true}`, string(output))
	require.Len(t, transport.requests, 1)
	require.Len(t, runner.calls, 2)
}

func TestRuntimeExecutorReturnsRunnerError(t *testing.T) {
	t.Parallel()

	transport := &fakeRoundTripper{
		statusCode: http.StatusOK,
		body:       "export async function invoke(payload) { return payload; }",
	}
	runner := &fakeRunner{
		err:    io.EOF,
		stderr: "deno failed",
	}

	executor := NewRuntimeExecutor(RuntimeExecutorConfig{
		HubURL:     "http://hub.example",
		CacheDir:   t.TempDir(),
		HTTPClient: &http.Client{Transport: transport},
		Runner:     runner,
	})

	_, err := executor.HandleJob(context.Background(), protocol.JobAssignMessage{
		JobID:       "job-1",
		ExtensionID: "ext-1",
		VersionID:   "ver-1",
		BundleToken: "bundle-token",
		Invocation:  json.RawMessage(`{"input":{}}`),
	})
	require.EqualError(t, err, "run deno bundle: deno failed")
}

type fakeRoundTripper struct {
	mu         sync.Mutex
	requests   []string
	statusCode int
	body       string
}

func (f *fakeRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	f.mu.Lock()
	f.requests = append(f.requests, request.URL.String())
	f.mu.Unlock()

	return &http.Response{
		StatusCode: f.statusCode,
		Status:     http.StatusText(f.statusCode),
		Body:       io.NopCloser(strings.NewReader(f.body)),
		Header:     make(http.Header),
	}, nil
}

type fakeRunner struct {
	mu     sync.Mutex
	calls  [][]string
	output string
	stderr string
	err    error
}

func (f *fakeRunner) Run(_ context.Context, _ string, args []string, stdout io.Writer, stderr io.Writer) error {
	f.mu.Lock()
	f.calls = append(f.calls, append([]string(nil), args...))
	f.mu.Unlock()

	if f.output != "" {
		_, _ = io.Copy(stdout, bytes.NewBufferString(f.output))
	}
	if f.stderr != "" {
		_, _ = io.Copy(stderr, bytes.NewBufferString(f.stderr))
	}

	return f.err
}
