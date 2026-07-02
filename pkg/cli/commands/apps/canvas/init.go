package canvas

import (
	"embed"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/superplanehq/superplane/pkg/cli/core"
)

//go:embed templates/*.yaml
var embeddedCanvasTemplates embed.FS

const blankCanvasYAML = `apiVersion: v1
kind: Canvas
metadata:
  name: "my-canvas"
spec:
  nodes: []
  edges: []
`

type embeddedCanvasTemplate struct {
	Name        string
	DisplayName string
	Description string
	Filename    string
}

var bakedInCanvasTemplates = []embeddedCanvasTemplate{
	{
		Name:        "health-check-monitor",
		DisplayName: "Health Check Monitor",
		Description: "Monitor any endpoint and mark when it goes down. The failure marker only runs when the endpoint transitions from healthy to failing, not on every failed check. No integrations required.",
		Filename:    "templates/health-check-monitor.yaml",
	},
}

type initCommand struct {
	template      *string
	listTemplates *bool
	outputFile    *string
}

func (c *initCommand) Execute(ctx core.CommandContext) error {
	if c.listTemplates != nil && *c.listTemplates {
		return c.executeListTemplates(ctx)
	}

	if c.template != nil && *c.template != "" {
		return c.executeTemplate(ctx, *c.template)
	}

	return c.executeBlank(ctx)
}

func (c *initCommand) executeBlank(ctx core.CommandContext) error {
	return c.output(ctx, []byte(blankCanvasYAML))
}

func (c *initCommand) executeTemplate(ctx core.CommandContext, name string) error {
	match, available, err := findEmbeddedCanvasTemplate(name)
	if err != nil {
		return err
	}
	if match == nil {
		return fmt.Errorf("template %q not found; available templates: %s", name, strings.Join(available, ", "))
	}

	data, err := embeddedCanvasTemplates.ReadFile(match.Filename)
	if err != nil {
		return fmt.Errorf("failed to read template %q: %w", match.Name, err)
	}

	return c.output(ctx, data)
}

func (c *initCommand) executeListTemplates(ctx core.CommandContext) error {
	if !ctx.Renderer.IsText() {
		summaries := make([]map[string]string, 0, len(bakedInCanvasTemplates))
		for _, template := range bakedInCanvasTemplates {
			summaries = append(summaries, map[string]string{
				"name":        template.DisplayName,
				"description": template.Description,
			})
		}
		return ctx.Renderer.Render(summaries)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		if len(bakedInCanvasTemplates) == 0 {
			_, err := fmt.Fprintln(stdout, "No templates found.")
			return err
		}

		writer := tabwriter.NewWriter(stdout, 0, 8, 2, ' ', 0)
		_, _ = fmt.Fprintln(writer, "NAME\tDESCRIPTION")
		for _, template := range bakedInCanvasTemplates {
			_, _ = fmt.Fprintf(writer, "%s\t%s\n", template.DisplayName, template.Description)
		}
		return writer.Flush()
	})
}

func (c *initCommand) output(ctx core.CommandContext, data []byte) error {
	if c.outputFile != nil && *c.outputFile != "" {
		return c.writeToFile(*c.outputFile, data)
	}

	_, err := ctx.Cmd.OutOrStdout().Write(data)
	return err
}

func (c *initCommand) writeToFile(path string, data []byte) error {
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("file %q already exists", path)
	}

	// #nosec
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Canvas file written to %s\n", path)
	return nil
}

func findEmbeddedCanvasTemplate(name string) (*embeddedCanvasTemplate, []string, error) {
	normalizedInput := normalizeTemplateName(name)
	available := make([]string, 0, len(bakedInCanvasTemplates))
	var match *embeddedCanvasTemplate

	for i, template := range bakedInCanvasTemplates {
		available = append(available, template.DisplayName)
		if strings.EqualFold(normalizeTemplateName(template.Name), normalizedInput) ||
			strings.EqualFold(normalizeTemplateName(template.DisplayName), normalizedInput) {
			match = &bakedInCanvasTemplates[i]
		}
	}

	return match, available, nil
}

// normalizeTemplateName converts a template name to a comparable form,
// so both "Health Check Monitor" and "health-check-monitor" match.
func normalizeTemplateName(name string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(name), " ", "-"))
}
