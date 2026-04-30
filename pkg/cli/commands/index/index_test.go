package index

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

const integrationsListResponse = `{
	"integrations": [
		{
			"name": "slack",
			"label": "Slack",
			"description": "Slack integration",
			"configuration": [{"name": "token", "type": "string", "required": true}],
			"capabilities": [
				{"type": "TYPE_ACTION", "name": "slack.post-message"},
				{"type": "TYPE_TRIGGER", "name": "slack.message-received"}
			]
		}
	]
}`

const actionsListResponse = `{
	"actions": [
		{
			"name": "http",
			"label": "HTTP",
			"description": "HTTP request action",
			"configuration": [{"name": "url", "type": "string", "required": true}],
			"exampleOutput": {"status": 200}
		}
	]
}`

const triggersListResponse = `{
	"triggers": [
		{
			"name": "cron",
			"label": "Cron",
			"description": "Cron trigger",
			"configuration": [{"name": "schedule", "type": "string", "required": true}],
			"exampleData": {"timestamp": "2025-01-01T00:00:00Z"}
		}
	]
}`

// Integrations tests

func TestIntegrationsListReturnsSummaryJSON(t *testing.T) {
	server := newIndexServer(t, "GET", "/api/v1/integrations", integrationsListResponse)
	ctx, stdout := newIndexCommandContext(t, server, "json")
	name := ""
	full := false
	cmd := &integrationsCommand{name: &name, full: &full}

	err := cmd.Execute(ctx)
	require.NoError(t, err)

	var result []map[string]string
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	require.Len(t, result, 1)
	require.Equal(t, "slack", result[0]["name"])
	require.Equal(t, "Slack", result[0]["label"])
	require.Equal(t, "Slack integration", result[0]["description"])

	raw := stdout.String()
	require.NotContains(t, raw, "token")
	require.NotContains(t, raw, "post-message")
}

func TestIntegrationsListReturnsFullJSON(t *testing.T) {
	server := newIndexServer(t, "GET", "/api/v1/integrations", integrationsListResponse)
	ctx, stdout := newIndexCommandContext(t, server, "json")
	name := ""
	full := true
	cmd := &integrationsCommand{name: &name, full: &full}

	err := cmd.Execute(ctx)
	require.NoError(t, err)

	raw := stdout.String()
	require.Contains(t, raw, "token")
	require.Contains(t, raw, "post-message")
	require.Contains(t, raw, "slack")
}

// Actions tests

func TestActionsListReturnsSummaryJSON(t *testing.T) {
	server := newIndexServer(t, "GET", "/api/v1/actions", actionsListResponse)
	ctx, stdout := newIndexCommandContext(t, server, "json")
	name := ""
	from := ""
	full := false
	cmd := &actionsCommand{name: &name, from: &from, full: &full}

	err := cmd.Execute(ctx)
	require.NoError(t, err)

	var result []map[string]string
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	require.Len(t, result, 1)
	require.Equal(t, "http", result[0]["name"])
	require.Equal(t, "HTTP", result[0]["label"])
	require.Equal(t, "HTTP request action", result[0]["description"])

	raw := stdout.String()
	require.NotContains(t, raw, "exampleOutput")
	require.NotContains(t, raw, "url")
}

func TestActionsListReturnsFullJSON(t *testing.T) {
	server := newIndexServer(t, "GET", "/api/v1/actions", actionsListResponse)
	ctx, stdout := newIndexCommandContext(t, server, "json")
	name := ""
	from := ""
	full := true
	cmd := &actionsCommand{name: &name, from: &from, full: &full}

	err := cmd.Execute(ctx)
	require.NoError(t, err)

	raw := stdout.String()
	require.Contains(t, raw, "url")
	require.Contains(t, raw, "http")
}

// Triggers tests

func TestTriggersListReturnsSummaryJSON(t *testing.T) {
	server := newIndexServer(t, "GET", "/api/v1/triggers", triggersListResponse)
	ctx, stdout := newIndexCommandContext(t, server, "json")
	name := ""
	from := ""
	full := false
	cmd := &triggersCommand{name: &name, from: &from, full: &full}

	err := cmd.Execute(ctx)
	require.NoError(t, err)

	var result []map[string]string
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	require.Len(t, result, 1)
	require.Equal(t, "cron", result[0]["name"])
	require.Equal(t, "Cron", result[0]["label"])
	require.Equal(t, "Cron trigger", result[0]["description"])

	raw := stdout.String()
	require.NotContains(t, raw, "schedule")
	require.NotContains(t, raw, "exampleData")
}

func TestTriggersListReturnsFullJSON(t *testing.T) {
	server := newIndexServer(t, "GET", "/api/v1/triggers", triggersListResponse)
	ctx, stdout := newIndexCommandContext(t, server, "json")
	name := ""
	from := ""
	full := true
	cmd := &triggersCommand{name: &name, from: &from, full: &full}

	err := cmd.Execute(ctx)
	require.NoError(t, err)

	raw := stdout.String()
	require.Contains(t, raw, "schedule")
	require.Contains(t, raw, "cron")
}

func TestIntegrationsListTextOutput(t *testing.T) {
	server := newIndexServer(t, "GET", "/api/v1/integrations", integrationsListResponse)
	ctx, stdout := newIndexCommandContext(t, server, "text")
	name := ""
	full := false
	cmd := &integrationsCommand{name: &name, full: &full}

	require.NoError(t, cmd.Execute(ctx))
	out := stdout.String()
	require.Contains(t, out, "slack")
	require.Contains(t, out, "Slack")
	require.Contains(t, out, "Slack integration")
	require.Contains(t, out, "NAME")
}

func TestIntegrationsDescribeByNameTextOutput(t *testing.T) {
	server := newIndexServer(t, "GET", "/api/v1/integrations", integrationsListResponse)
	ctx, stdout := newIndexCommandContext(t, server, "text")
	name := "slack"
	full := false
	cmd := &integrationsCommand{name: &name, full: &full}

	require.NoError(t, cmd.Execute(ctx))
	out := stdout.String()
	require.Contains(t, out, "Name: slack")
	require.Contains(t, out, "Configuration:")
	require.Contains(t, out, "Actions:")
	require.Contains(t, out, "slack.post-message")
	require.Contains(t, out, "Triggers:")
	require.Contains(t, out, "slack.message-received")
}

func TestActionsListTextOutput(t *testing.T) {
	server := newIndexServer(t, "GET", "/api/v1/actions", actionsListResponse)
	ctx, stdout := newIndexCommandContext(t, server, "text")
	name := ""
	from := ""
	full := false
	cmd := &actionsCommand{name: &name, from: &from, full: &full}

	require.NoError(t, cmd.Execute(ctx))
	out := stdout.String()
	require.Contains(t, out, "NAME")
	require.Contains(t, out, "http")
	require.Contains(t, out, "HTTP")
}

func TestActionsListFromIntegrationTextOutput(t *testing.T) {
	server := newMultiRouteIndexServer(t, map[string]string{
		"/api/v1/integrations": integrationsListResponse,
	})
	ctx, stdout := newIndexCommandContext(t, server, "text")
	name := ""
	from := "slack"
	full := false
	cmd := &actionsCommand{name: &name, from: &from, full: &full}

	require.NoError(t, cmd.Execute(ctx))
	out := stdout.String()
	require.Contains(t, out, "slack.post-message")
}

func TestActionsDescribeByNameTextOutput(t *testing.T) {
	const describeHTTP = `{
		"action": {
			"name": "http",
			"label": "HTTP",
			"description": "HTTP request",
			"outputChannels": [
				{"name": "", "label": "Default", "description": ""},
				{"name": "success", "label": "success", "description": "When the request succeeds"},
				{"name": "plain", "label": "", "description": ""}
			],
			"configuration": [{"name": "url", "type": "string", "required": true, "description": "Endpoint URL"}],
			"exampleOutput": {"status": 200, "body": "ok"}
		}
	}`
	server := newIndexServer(t, "GET", "/api/v1/actions/http", describeHTTP)
	ctx, stdout := newIndexCommandContext(t, server, "text")
	name := "http"
	from := ""
	full := false
	cmd := &actionsCommand{name: &name, from: &from, full: &full}

	require.NoError(t, cmd.Execute(ctx))
	out := stdout.String()
	require.Contains(t, out, "Name: http")
	require.Contains(t, out, "Output Channels:")
	require.Contains(t, out, "(unnamed)")
	require.Contains(t, out, "When the request succeeds")
	require.Contains(t, out, "Configuration:")
	require.Contains(t, out, "url")
	require.Contains(t, out, "Example Payload:")
	require.Contains(t, out, `"status": 200`)
}

func TestTriggersListTextOutput(t *testing.T) {
	server := newIndexServer(t, "GET", "/api/v1/triggers", triggersListResponse)
	ctx, stdout := newIndexCommandContext(t, server, "text")
	name := ""
	from := ""
	full := false
	cmd := &triggersCommand{name: &name, from: &from, full: &full}

	require.NoError(t, cmd.Execute(ctx))
	out := stdout.String()
	require.Contains(t, out, "cron")
	require.Contains(t, out, "Cron")
}

func TestTriggersListFromIntegrationTextOutput(t *testing.T) {
	server := newMultiRouteIndexServer(t, map[string]string{
		"/api/v1/integrations": integrationsListResponse,
	})
	ctx, stdout := newIndexCommandContext(t, server, "text")
	name := ""
	from := "slack"
	full := false
	cmd := &triggersCommand{name: &name, from: &from, full: &full}

	require.NoError(t, cmd.Execute(ctx))
	require.Contains(t, stdout.String(), "slack.message-received")
}

func TestTriggersDescribeByNameTextOutput(t *testing.T) {
	const describeCron = `{
		"trigger": {
			"name": "cron",
			"label": "Cron",
			"description": "On a schedule",
			"configuration": [{"name": "schedule", "type": "string", "required": true, "description": "Cron expr"}],
			"exampleData": {"tick": true}
		}
	}`
	server := newIndexServer(t, "GET", "/api/v1/triggers/cron", describeCron)
	ctx, stdout := newIndexCommandContext(t, server, "text")
	name := "cron"
	from := ""
	full := false
	cmd := &triggersCommand{name: &name, from: &from, full: &full}

	require.NoError(t, cmd.Execute(ctx))
	out := stdout.String()
	require.Contains(t, out, "Name: cron")
	require.Contains(t, out, "schedule")
	require.Contains(t, out, "Example Payload:")
	require.Contains(t, out, `"tick": true`)
}

// Helpers

func newIndexServer(t *testing.T, method, path, response string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, method, r.Method)
		require.Equal(t, path, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(response))
	}))
	t.Cleanup(server.Close)
	return server
}

func newIndexCommandContext(
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

	return core.CommandContext{
		Context:  context.Background(),
		Cmd:      cobraCmd,
		API:      openapi_client.NewAPIClient(config),
		Renderer: renderer,
	}, stdout
}
