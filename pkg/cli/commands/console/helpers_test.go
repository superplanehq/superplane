package console

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

// fakeConfig provides the canvas-active accessor expected by
// core.ResolveCanvasID. CLI tests typically pass `--canvas-id` directly
// (so the active canvas is irrelevant), but a few exercises rely on the
// fallback behavior — set `activeCanvas` for those cases.
type fakeConfig struct {
	activeCanvas string
	url          string
}

func (f *fakeConfig) GetActiveCanvas() string               { return f.activeCanvas }
func (f *fakeConfig) SetActiveCanvas(canvasID string) error { return nil }
func (f *fakeConfig) GetURL() string                        { return f.url }

type requestExpectation struct {
	method string
	path   string
	handle func(t *testing.T, w http.ResponseWriter, r *http.Request)
}

type apiTestServer struct {
	t            *testing.T
	expectations []requestExpectation
	calls        []string
	server       *httptest.Server
}

// newAPITestServer creates a strict mock server: each request must match
// the next expectation in order. Tests that don't care about call order
// should pass the routes via newRouteAPITestServer instead.
func newAPITestServer(t *testing.T, expectations ...requestExpectation) *apiTestServer {
	t.Helper()

	s := &apiTestServer{t: t, expectations: expectations}
	s.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.calls = append(s.calls, r.Method+" "+r.URL.Path)

		if len(s.expectations) == 0 {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		next := s.expectations[0]
		require.Equal(t, next.method, r.Method)
		require.Equal(t, next.path, r.URL.Path)

		s.expectations = s.expectations[1:]
		if next.handle != nil {
			next.handle(t, w, r)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	t.Cleanup(s.server.Close)
	return s
}

// newRouteAPITestServer returns a server that responds based on the URL
// path, regardless of order. Useful when commands fan out (e.g. the data
// command may fetch the dashboard plus list memory/runs/events).
func newRouteAPITestServer(t *testing.T, routes map[string]string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response, ok := routes[r.URL.Path]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(response))
	}))
	t.Cleanup(server.Close)
	return server
}

func newCommandContext(
	t *testing.T,
	server *httptest.Server,
	outputFormat string,
) (core.CommandContext, *bytes.Buffer) {
	t.Helper()

	stdout := bytes.NewBuffer(nil)
	renderer, err := core.NewRenderer(outputFormat, stdout)
	require.NoError(t, err)

	cobraCmd := &cobra.Command{}
	cobraCmd.SetOut(stdout)
	cobraCmd.SetIn(strings.NewReader(""))

	ctx := core.CommandContext{
		Context:  context.Background(),
		Cmd:      cobraCmd,
		Renderer: renderer,
		Config:   &fakeConfig{},
	}

	if server != nil {
		config := openapi_client.NewConfiguration()
		config.Servers = openapi_client.ServerConfigurations{{URL: server.URL}}
		ctx.API = openapi_client.NewAPIClient(config)
	}

	return ctx, stdout
}

// stringPtr returns a pointer to the given string. Test helper that lets
// us pass cobra-style flag fields without declaring a local variable for
// every flag.
func stringPtr(s string) *string {
	return &s
}

// boolPtr returns a pointer to the given bool, mirroring stringPtr.
func boolPtr(b bool) *bool {
	return &b
}

// int64Ptr returns a pointer to the given int64.
func int64Ptr(i int64) *int64 {
	return &i
}
