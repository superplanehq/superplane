package common

import (
	"bytes"
	"context"
	"net/http/httptest"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

func NewCreateCommandContextForTest(t *testing.T, server *httptest.Server, outputFormat string) (core.CommandContext, *bytes.Buffer) {
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

func NewCreateCommandContextWithConfigForTest(
	t *testing.T,
	server *httptest.Server,
	outputFormat string,
	config core.ConfigContext,
) (core.CommandContext, *bytes.Buffer) {
	t.Helper()

	ctx, stdout := NewCreateCommandContextForTest(t, server, outputFormat)
	ctx.Config = config
	return ctx, stdout
}
