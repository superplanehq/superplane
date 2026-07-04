package canvas

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

func newInitCommandContext(t *testing.T, outputFormat string) (core.CommandContext, *bytes.Buffer) {
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

	return ctx, stdout
}

func TestInitBlankCanvasOutputsValidYAML(t *testing.T) {
	ctx, stdout := newInitCommandContext(t, "text")
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

func TestInitWithTemplateOutputsEmbeddedContent(t *testing.T) {
	ctx, stdout := newInitCommandContext(t, "text")
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
	require.Contains(t, output, "schedule-trigger-001")
	require.Contains(t, output, "http-check-002")
	require.NotContains(t, output, "isTemplate")
}

func TestInitWithTemplateSlugOutputsEmbeddedContent(t *testing.T) {
	ctx, stdout := newInitCommandContext(t, "text")
	template := "health-check-monitor"
	listTemplates := false
	outputFile := ""

	err := (&initCommand{
		template:      &template,
		listTemplates: &listTemplates,
		outputFile:    &outputFile,
	}).Execute(ctx)

	require.NoError(t, err)
	require.Contains(t, stdout.String(), "Health Check Monitor")
}

func TestInitWithUnknownTemplateReturnsError(t *testing.T) {
	ctx, _ := newInitCommandContext(t, "text")
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

func TestInitListTemplatesShowsEmbeddedTemplates(t *testing.T) {
	ctx, stdout := newInitCommandContext(t, "text")
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
	require.NotContains(t, output, "Staged Release")
}

func TestInitListTemplatesJSON(t *testing.T) {
	ctx, stdout := newInitCommandContext(t, "json")
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
	require.NotContains(t, output, "Staged Release")
}

func TestInitWritesToFileWithOutputFlag(t *testing.T) {
	ctx, _ := newInitCommandContext(t, "text")
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
	ctx, _ := newInitCommandContext(t, "text")
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
	ctx, _ := newInitCommandContext(t, "text")
	template := "health-check-monitor"
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
	require.Contains(t, content, "schedule-trigger-001")
}
