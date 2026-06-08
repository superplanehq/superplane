package cli

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

// FakeConfig is a test stub implementation of the CLI's core.ConfigContext.
// The app-active accessors are no-ops because the app commands under test
// here do not use them.
type FakeConfig struct {
	URL       string
	ActiveApp string
}

func (f *FakeConfig) GetActiveApp() string {
	if f.ActiveApp != "" {
		return f.ActiveApp
	}
	return ""
}

func (f *FakeConfig) SetActiveApp(appID string) error {
	return nil
}

func (f *FakeConfig) GetURL() string {
	return f.URL
}

func NewCommandContext(t *testing.T, server *httptest.Server, outputFormat string) (core.CommandContext, *bytes.Buffer) {
	t.Helper()

	stdout := bytes.NewBuffer(nil)
	renderer, err := core.NewRenderer(outputFormat, stdout)
	require.NoError(t, err)

	cobraCmd := &cobra.Command{}
	cobraCmd.SetOut(stdout)

	commandCtx := core.CommandContext{
		Context:  context.Background(),
		Cmd:      cobraCmd,
		Renderer: renderer,
	}

	if server != nil {
		config := openapi_client.NewConfiguration()
		config.Servers = openapi_client.ServerConfigurations{
			{URL: server.URL},
		}
		commandCtx.API = openapi_client.NewAPIClient(config)
	}

	return commandCtx, stdout
}

func NewCommandContextWithConfig(
	t *testing.T,
	server *httptest.Server,
	outputFormat string,
	config core.ConfigContext,
) (core.CommandContext, *bytes.Buffer) {
	t.Helper()

	ctx, stdout := NewCommandContext(t, server, outputFormat)
	ctx.Config = config
	return ctx, stdout
}

func NewCommandContextWithRedirectPolicy(
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

	config := openapi_client.NewConfiguration()
	config.Servers = openapi_client.ServerConfigurations{
		{URL: server.URL},
	}
	config.HTTPClient = &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) > 0 && req.Method != via[0].Method {
				return fmt.Errorf(
					"refusing to follow redirect that changes method from %s to %s",
					via[0].Method, req.Method,
				)
			}
			return nil
		},
	}

	return core.CommandContext{
		Context:  context.Background(),
		Cmd:      cobraCmd,
		API:      openapi_client.NewAPIClient(config),
		Renderer: renderer,
	}, stdout
}
