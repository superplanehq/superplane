package integrations

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

const meResponse = `{
	"user": {
		"id": "user-1",
		"organizationId": "org-1"
	}
}`

// Connected integrations returned in scrambled order across two providers,
// to prove the CLI sorts them by integrationName then metadata.name.
const connectedIntegrationsResponse = `{
	"integrations": [
		{"metadata": {"id": "1", "name": "github-superplanehq"}, "spec": {"integrationName": "github"}, "status": {"state": "ready"}},
		{"metadata": {"id": "2", "name": "daytona-staging"},     "spec": {"integrationName": "daytona"}, "status": {"state": "ready"}},
		{"metadata": {"id": "3", "name": "github-acme"},         "spec": {"integrationName": "github"}, "status": {"state": "ready"}},
		{"metadata": {"id": "4", "name": "daytona-prod"},        "spec": {"integrationName": "daytona"}, "status": {"state": "ready"}}
	]
}`

const availableIntegrationsResponse = `{
	"integrations": [
		{"name": "github",  "label": "GitHub",  "description": "GitHub integration"},
		{"name": "daytona", "label": "Daytona", "description": "Daytona integration"}
	]
}`

func newIntegrationsListServer(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/api/v1/me":
			_, _ = w.Write([]byte(meResponse))
		case r.URL.Path == "/api/v1/organizations/org-1/integrations":
			_, _ = w.Write([]byte(connectedIntegrationsResponse))
		case r.URL.Path == "/api/v1/integrations":
			_, _ = w.Write([]byte(availableIntegrationsResponse))
		default:
			t.Fatalf("unexpected request: %s", r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func newListCommandContextForTest(t *testing.T, server *httptest.Server, outputFormat string) (core.CommandContext, *bytes.Buffer) {
	t.Helper()

	stdout := bytes.NewBuffer(nil)
	renderer, err := core.NewRenderer(outputFormat, stdout)
	require.NoError(t, err)

	cobraCmd := &cobra.Command{}
	cobraCmd.SetOut(stdout)

	ctx := core.CommandContext{
		Context:  context.Background(),
		Cmd:      cobraCmd,
		Renderer: renderer,
	}

	config := openapi_client.NewConfiguration()
	config.Servers = openapi_client.ServerConfigurations{{URL: server.URL}}
	ctx.API = openapi_client.NewAPIClient(config)

	return ctx, stdout
}

func TestListCommandSortsByIntegrationNameThenName(t *testing.T) {
	server := newIntegrationsListServer(t)
	ctx, stdout := newListCommandContextForTest(t, server, "json")

	require.NoError(t, (&listCommand{}).Execute(ctx))

	var result []struct {
		Metadata struct{ ID, Name string } `json:"metadata"`
		Spec     struct {
			IntegrationName string `json:"integrationName"`
		} `json:"spec"`
	}
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	require.Len(t, result, 4)

	// Expected order: daytona group first (alphabetical by integrationName),
	// each group internally sorted by metadata.name.
	require.Equal(t, "daytona", result[0].Spec.IntegrationName)
	require.Equal(t, "daytona-prod", result[0].Metadata.Name)
	require.Equal(t, "daytona", result[1].Spec.IntegrationName)
	require.Equal(t, "daytona-staging", result[1].Metadata.Name)
	require.Equal(t, "github", result[2].Spec.IntegrationName)
	require.Equal(t, "github-acme", result[2].Metadata.Name)
	require.Equal(t, "github", result[3].Spec.IntegrationName)
	require.Equal(t, "github-superplanehq", result[3].Metadata.Name)
}

func TestListCommandTextOutputGroupsSameProvider(t *testing.T) {
	server := newIntegrationsListServer(t)
	ctx, stdout := newListCommandContextForTest(t, server, "text")

	require.NoError(t, (&listCommand{}).Execute(ctx))

	lines := strings.Split(strings.TrimRight(stdout.String(), "\n"), "\n")
	require.Len(t, lines, 5) // header + 4 rows

	// The two daytona rows must appear before the two github rows.
	require.Contains(t, lines[1], "daytona-prod")
	require.Contains(t, lines[2], "daytona-staging")
	require.Contains(t, lines[3], "github-acme")
	require.Contains(t, lines[4], "github-superplanehq")
}
