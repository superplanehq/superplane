package widgets

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

type fakeConfig struct {
	activeCanvas string
}

func (f *fakeConfig) GetActiveCanvas() string               { return f.activeCanvas }
func (f *fakeConfig) SetActiveCanvas(canvasID string) error { return nil }
func (f *fakeConfig) GetURL() string                        { return "" }

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

func stringPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}

func int32Ptr(i int32) *int32 {
	return &i
}
