package console

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type requestExpectation struct {
	method string
	path   string
	handle func(t *testing.T, w http.ResponseWriter, r *http.Request)
}

// fakeConfig is a minimal `core.ConfigContext` test double that returns
// a preconfigured active canvas. Only `GetActiveApp` is consulted by
// the console commands; the rest are stubs that satisfy the interface.
type fakeConfig struct {
	activeApp string
}

func (f *fakeConfig) GetActiveApp() string      { return f.activeApp }
func (f *fakeConfig) SetActiveApp(string) error { return nil }
func (f *fakeConfig) GetURL() string            { return "" }

type apiTestServer struct {
	t            *testing.T
	expectations []requestExpectation
	calls        []string
	server       *httptest.Server
}

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

func (s *apiTestServer) AssertCalls(t *testing.T, calls []string) {
	t.Helper()
	require.Equal(t, calls, s.calls)
	require.Len(t, s.expectations, 0, "unused request expectations")
}

// newConsoleCommandContext builds a command context wired to the given
// test HTTP server. Stdin is left as an empty buffer so commands that
// resolve YAML from stdin do not block on tty I/O.
func newConsoleCommandContext(
	t *testing.T,
	server *httptest.Server,
	outputFormat string,
	stdin io.Reader,
) (core.CommandContext, *bytes.Buffer) {
	t.Helper()

	stdout := bytes.NewBuffer(nil)
	renderer, err := core.NewRenderer(outputFormat, stdout)
	require.NoError(t, err)

	cobraCmd := &cobra.Command{}
	cobraCmd.SetOut(stdout)
	if stdin == nil {
		cobraCmd.SetIn(bytes.NewReader(nil))
	} else {
		cobraCmd.SetIn(stdin)
	}

	commandCtx := core.CommandContext{
		Context:  context.Background(),
		Cmd:      cobraCmd,
		Renderer: renderer,
	}

	if server != nil {
		config := openapi_client.NewConfiguration()
		config.Servers = openapi_client.ServerConfigurations{{URL: server.URL}}
		commandCtx.API = openapi_client.NewAPIClient(config)
	}

	return commandCtx, stdout
}
