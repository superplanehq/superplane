package index

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

func newMultiRouteIndexServer(t *testing.T, routes map[string]string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
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

func TestDumpWritesFullRegistryToFile(t *testing.T) {
	server := newMultiRouteIndexServer(t, map[string]string{
		"/api/v1/integrations": integrationsListResponse,
		"/api/v1/actions":      actionsListResponse,
		"/api/v1/triggers":     triggersListResponse,
	})

	outFile := filepath.Join(t.TempDir(), "index.json")
	ctx, stdout := newIndexCommandContext(t, server, "text")
	cmd := &dumpCommand{file: &outFile}

	err := cmd.Execute(ctx)
	require.NoError(t, err)

	// The message printed to stdout names the output file.
	require.Contains(t, stdout.String(), outFile)
	require.Contains(t, stdout.String(), "Index downloaded to")

	// The file must exist and be valid JSON with all three sections.
	raw, err := os.ReadFile(outFile)
	require.NoError(t, err)

	var dump indexDump
	require.NoError(t, json.Unmarshal(raw, &dump))

	require.Len(t, dump.Integrations, 1)
	require.Equal(t, "slack", dump.Integrations[0].GetName())

	require.Len(t, dump.Actions, 1)
	require.Equal(t, "http", dump.Actions[0].GetName())

	require.Len(t, dump.Triggers, 1)
	require.Equal(t, "cron", dump.Triggers[0].GetName())
}

func TestDumpIntegrationContainsNestedFields(t *testing.T) {
	server := newMultiRouteIndexServer(t, map[string]string{
		"/api/v1/integrations": integrationsListResponse,
		"/api/v1/actions":      actionsListResponse,
		"/api/v1/triggers":     triggersListResponse,
	})

	outFile := filepath.Join(t.TempDir(), "index.json")
	ctx, _ := newIndexCommandContext(t, server, "text")
	cmd := &dumpCommand{file: &outFile}

	require.NoError(t, cmd.Execute(ctx))

	raw, err := os.ReadFile(outFile)
	require.NoError(t, err)

	// Integration nested data (components, triggers, config) must be present.
	rawStr := string(raw)
	require.Contains(t, rawStr, "post-message")
	require.Contains(t, rawStr, "message-received")
	require.Contains(t, rawStr, "token")
}

func TestDumpFullComponentFields(t *testing.T) {
	server := newMultiRouteIndexServer(t, map[string]string{
		"/api/v1/integrations": integrationsListResponse,
		"/api/v1/actions":      actionsListResponse,
		"/api/v1/triggers":     triggersListResponse,
	})

	outFile := filepath.Join(t.TempDir(), "index.json")
	ctx, _ := newIndexCommandContext(t, server, "text")
	cmd := &dumpCommand{file: &outFile}

	require.NoError(t, cmd.Execute(ctx))

	raw, err := os.ReadFile(outFile)
	require.NoError(t, err)

	var dump indexDump
	require.NoError(t, json.Unmarshal(raw, &dump))

	component := dump.Actions[0]
	require.Equal(t, "HTTP", component.GetLabel())

	config := component.GetConfiguration()
	require.Len(t, config, 1)
	require.Equal(t, "url", config[0].GetName())
	require.True(t, config[0].GetRequired())

	exampleOutput := component.GetExampleOutput()
	require.NotEmpty(t, exampleOutput)
}

func TestDumpFullTriggerFields(t *testing.T) {
	server := newMultiRouteIndexServer(t, map[string]string{
		"/api/v1/integrations": integrationsListResponse,
		"/api/v1/actions":      actionsListResponse,
		"/api/v1/triggers":     triggersListResponse,
	})

	outFile := filepath.Join(t.TempDir(), "index.json")
	ctx, _ := newIndexCommandContext(t, server, "text")
	cmd := &dumpCommand{file: &outFile}

	require.NoError(t, cmd.Execute(ctx))

	raw, err := os.ReadFile(outFile)
	require.NoError(t, err)

	var dump indexDump
	require.NoError(t, json.Unmarshal(raw, &dump))

	trigger := dump.Triggers[0]
	require.Equal(t, "Cron", trigger.GetLabel())

	config := trigger.GetConfiguration()
	require.Len(t, config, 1)
	require.Equal(t, "schedule", config[0].GetName())

	exampleData := trigger.GetExampleData()
	require.NotEmpty(t, exampleData)
}

func TestDumpDefaultFileIsInTempDir(t *testing.T) {
	cmd := newDumpCommand(core.BindOptions{})
	flag := cmd.Flags().Lookup("file")
	require.NotNil(t, flag)
	expected := filepath.Join(os.TempDir(), "superplane-index.json")
	require.Equal(t, expected, flag.DefValue)
}

func TestDumpErrorOnAPIFailure(t *testing.T) {
	// Server only handles the integrations route; components call will get 404.
	server := newMultiRouteIndexServer(t, map[string]string{
		"/api/v1/integrations": integrationsListResponse,
	})

	outFile := filepath.Join(t.TempDir(), "index.json")
	ctx, _ := newIndexCommandContext(t, server, "text")
	cmd := &dumpCommand{file: &outFile}

	err := cmd.Execute(ctx)
	require.Error(t, err)

	// The file must not have been written on error.
	_, statErr := os.Stat(outFile)
	require.True(t, os.IsNotExist(statErr))
}

func TestDumpErrorOnUnwritablePath(t *testing.T) {
	server := newMultiRouteIndexServer(t, map[string]string{
		"/api/v1/integrations": integrationsListResponse,
		"/api/v1/actions":      actionsListResponse,
		"/api/v1/triggers":     triggersListResponse,
	})

	// A directory that doesn't exist cannot be written to.
	outFile := "/nonexistent-dir/superplane-index.json"
	ctx, _ := newIndexCommandContext(t, server, "text")
	cmd := &dumpCommand{file: &outFile}

	err := cmd.Execute(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "writing to")
}

func TestDumpUsesProvidedOpenAPITypes(t *testing.T) {
	server := newMultiRouteIndexServer(t, map[string]string{
		"/api/v1/integrations": integrationsListResponse,
		"/api/v1/actions":      actionsListResponse,
		"/api/v1/triggers":     triggersListResponse,
	})

	outFile := filepath.Join(t.TempDir(), "index.json")
	ctx, _ := newIndexCommandContext(t, server, "text")
	cmd := &dumpCommand{file: &outFile}

	require.NoError(t, cmd.Execute(ctx))

	raw, err := os.ReadFile(outFile)
	require.NoError(t, err)

	// Verify the dump uses full openapi types (not summary maps).
	var dump indexDump
	require.NoError(t, json.Unmarshal(raw, &dump))

	var integration openapi_client.IntegrationsIntegrationDefinition
	require.IsType(t, integration, dump.Integrations[0])

	var action openapi_client.SuperplaneActionsAction
	require.IsType(t, action, dump.Actions[0])

	var trigger openapi_client.TriggersTrigger
	require.IsType(t, trigger, dump.Triggers[0])
}
