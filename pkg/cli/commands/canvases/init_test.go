package canvases

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

const templatesListResponse = `{
	"canvases": [
		{
			"metadata": {
				"id": "tpl-001",
				"name": "Health Check Monitor",
				"description": "Monitor any endpoint and get notified when it goes down.",
				"isTemplate": true
			},
			"spec": {
				"nodes": [
					{"id": "trigger-001", "name": "Check every 10 minutes", "type": "TYPE_TRIGGER"},
					{"id": "http-001", "name": "Health check request", "type": "TYPE_COMPONENT"}
				],
				"edges": [
					{"sourceId": "trigger-001", "targetId": "http-001", "channel": "default"}
				]
			}
		},
		{
			"metadata": {
				"id": "tpl-002",
				"name": "Staged Release",
				"description": "Gradually roll out releases through stages.",
				"isTemplate": true
			},
			"spec": {
				"nodes": [
					{"id": "trigger-002", "name": "On release", "type": "TYPE_TRIGGER"}
				],
				"edges": []
			}
		},
		{
			"metadata": {
				"id": "canvas-001",
				"name": "My Canvas",
				"description": "A regular canvas",
				"isTemplate": false
			},
			"spec": {
				"nodes": [],
				"edges": []
			}
		}
	]
}`

func newTemplatesServer(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/api/v1/canvases", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(templatesListResponse))
	}))
	t.Cleanup(server.Close)
	return server
}

func newInitCommandContext(t *testing.T, server *httptest.Server, outputFormat string) (core.CommandContext, *bytes.Buffer) {
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

	if server != nil {
		config := openapi_client.NewConfiguration()
		config.Servers = openapi_client.ServerConfigurations{
			{URL: server.URL},
		}
		ctx.API = openapi_client.NewAPIClient(config)
	}

	return ctx, stdout
}

func TestInitBlankCanvasOutputsValidYAML(t *testing.T) {
	ctx, stdout := newInitCommandContext(t, nil, "text")
	template := ""
	listTemplates := false
	outputFile := ""

	err := (&initCommand{
		template:      &template,
		listTemplates: &listTemplates,
		outputFile:    &outputFile,
	}).Execute(ctx)

	require.NoError(t, err)

	output := stdout.String()
	require.Contains(t, output, "apiVersion: v1")
	require.Contains(t, output, "kind: Canvas")
	require.Contains(t, output, `name: "my-canvas"`)
	require.Contains(t, output, "nodes: []")
	require.Contains(t, output, "edges: []")
}

func TestInitWithTemplateOutputsTemplateContent(t *testing.T) {
	server := newTemplatesServer(t)
	ctx, stdout := newInitCommandContext(t, server, "text")
	template := "Health Check Monitor"
	listTemplates := false
	outputFile := ""

	err := (&initCommand{
		template:      &template,
		listTemplates: &listTemplates,
		outputFile:    &outputFile,
	}).Execute(ctx)

	require.NoError(t, err)

	output := stdout.String()
	require.Contains(t, output, "Health Check Monitor")
	require.Contains(t, output, "trigger-001")
	require.Contains(t, output, "http-001")

	// Should not contain server-only fields
	require.NotContains(t, output, "tpl-001")
	require.NotContains(t, output, "isTemplate")
}

}

func TestInitWithInvalidTemplateReturnsError(t *testing.T) {
	server := newTemplatesServer(t)
	ctx, _ := newInitCommandContext(t, server, "text")
	template := "nonexistent"
	listTemplates := false
	outputFile := ""

	err := (&initCommand{
		template:      &template,
		listTemplates: &listTemplates,
		outputFile:    &outputFile,
	}).Execute(ctx)

	require.Error(t, err)
	require.Contains(t, err.Error(), `template "nonexistent" not found`)
	require.Contains(t, err.Error(), "Health Check Monitor")
}

func TestInitListTemplatesShowsOnlyTemplates(t *testing.T) {
	server := newTemplatesServer(t)
	ctx, stdout := newInitCommandContext(t, server, "text")
	template := ""
	listTemplates := true
	outputFile := ""

	err := (&initCommand{
		template:      &template,
		listTemplates: &listTemplates,
		outputFile:    &outputFile,
	}).Execute(ctx)

	require.NoError(t, err)

	output := stdout.String()
	require.Contains(t, output, "Health Check Monitor")
	require.Contains(t, output, "Staged Release")
	require.NotContains(t, output, "My Canvas")
}

func TestInitListTemplatesJSON(t *testing.T) {
	server := newTemplatesServer(t)
	ctx, stdout := newInitCommandContext(t, server, "json")
	template := ""
	listTemplates := true
	outputFile := ""

	err := (&initCommand{
		template:      &template,
		listTemplates: &listTemplates,
		outputFile:    &outputFile,
	}).Execute(ctx)

	require.NoError(t, err)

	output := stdout.String()
	require.Contains(t, output, `"name": "Health Check Monitor"`)
	require.Contains(t, output, `"name": "Staged Release"`)
	require.NotContains(t, output, "My Canvas")
}

func TestInitWritesToFileWithOutputFlag(t *testing.T) {
	ctx, _ := newInitCommandContext(t, nil, "text")
	template := ""
	listTemplates := false
	outputPath := filepath.Join(t.TempDir(), "canvas.yaml")

	err := (&initCommand{
		template:      &template,
		listTemplates: &listTemplates,
		outputFile:    &outputPath,
	}).Execute(ctx)

	require.NoError(t, err)

	data, err := os.ReadFile(outputPath)
	require.NoError(t, err)

	content := string(data)
	require.Contains(t, content, "apiVersion: v1")
	require.Contains(t, content, "kind: Canvas")
	require.Contains(t, content, `name: "my-canvas"`)
}

func TestInitErrorsWhenOutputFileExists(t *testing.T) {
	ctx, _ := newInitCommandContext(t, nil, "text")
	template := ""
	listTemplates := false
	outputPath := filepath.Join(t.TempDir(), "canvas.yaml")

	err := os.WriteFile(outputPath, []byte("existing content"), 0644)
	require.NoError(t, err)

	err = (&initCommand{
		template:      &template,
		listTemplates: &listTemplates,
		outputFile:    &outputPath,
	}).Execute(ctx)

	require.Error(t, err)
	require.Contains(t, err.Error(), "already exists")

	data, err := os.ReadFile(outputPath)
	require.NoError(t, err)
	require.Equal(t, "existing content", string(data))
}

func TestInitTemplateToFile(t *testing.T) {
	server := newTemplatesServer(t)
	ctx, _ := newInitCommandContext(t, server, "text")
	template := "Health Check Monitor"
	listTemplates := false
	outputPath := filepath.Join(t.TempDir(), "hc.yaml")

	err := (&initCommand{
		template:      &template,
		listTemplates: &listTemplates,
		outputFile:    &outputPath,
	}).Execute(ctx)

	require.NoError(t, err)

	data, err := os.ReadFile(outputPath)
	require.NoError(t, err)

	content := string(data)
	require.Contains(t, content, "Health Check Monitor")
	require.Contains(t, content, "trigger-001")
}
