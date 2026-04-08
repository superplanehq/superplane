package canvases

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
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

	return core.CommandContext{
		Context:  context.Background(),
		Cmd:      cobraCmd,
		Renderer: renderer,
	}, stdout
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

func TestInitWithTemplateOutputsTemplateContent(t *testing.T) {
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

	output := stdout.String()
	require.Contains(t, output, "apiVersion: v1")
	require.Contains(t, output, "kind: Canvas")
	require.Contains(t, output, `name: "Health Check Monitor"`)
	require.Contains(t, output, "TYPE_TRIGGER")
	require.Contains(t, output, "TYPE_COMPONENT")
}

func TestInitWithInvalidTemplateReturnsError(t *testing.T) {
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
	require.Contains(t, err.Error(), "health-check-monitor")
}

func TestInitListTemplatesShowsAllTemplates(t *testing.T) {
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
	require.Contains(t, output, "health-check-monitor")
	require.Contains(t, output, "automated-rollback")
	require.Contains(t, output, "staged-release")
	require.Contains(t, output, "incident-router")
	require.Contains(t, output, "incident-data-collection")
	require.Contains(t, output, "policy-gated-deployment")
	require.Contains(t, output, "multi-repo-ci-and-release")
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
	require.Contains(t, output, `"name": "health-check-monitor"`)
	require.Contains(t, output, `"title": "Health Check Monitor"`)
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
	require.Contains(t, content, "apiVersion: v1")
	require.Contains(t, content, `name: "Health Check Monitor"`)
}

func TestLoadTemplateEntries(t *testing.T) {
	entries, err := loadTemplateEntries()
	require.NoError(t, err)
	require.NotEmpty(t, entries)

	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.key)
		require.NotEmpty(t, e.name, "template %s should have a name", e.key)
		require.NotEmpty(t, e.description, "template %s should have a description", e.key)
	}

	require.True(t, contains(names, "health-check-monitor"))
	require.True(t, contains(names, "automated-rollback"))
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if strings.EqualFold(s, item) {
			return true
		}
	}
	return false
}
