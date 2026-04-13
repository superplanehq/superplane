package canvases

import (
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/ghodss/yaml"
	"github.com/superplanehq/superplane/pkg/cli/commands/canvases/models"
	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

const blankCanvasYAML = `apiVersion: v1
kind: Canvas
metadata:
  name: "my-canvas"
spec:
  nodes: []
  edges: []
`

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
	templates, err := fetchTemplates(ctx)
	if err != nil {
		return err
	}

	var match *openapi_client.CanvasesCanvas
	var available []string
	normalizedInput := normalizeTemplateName(name)
	for i, t := range templates {
		metadata := t.GetMetadata()
		templateName := metadata.GetName()
		available = append(available, templateName)
		if strings.EqualFold(normalizeTemplateName(templateName), normalizedInput) {
			match = &templates[i]
		}
	}

	if match == nil {
		return fmt.Errorf("template %q not found; available templates: %s", name, strings.Join(available, ", "))
	}

	resource := templateResourceFromCanvas(*match)
	data, err := yaml.Marshal(resource)
	if err != nil {
		return fmt.Errorf("failed to marshal template: %w", err)
	}

	return c.output(ctx, data)
}

func (c *initCommand) executeListTemplates(ctx core.CommandContext) error {
	templates, err := fetchTemplates(ctx)
	if err != nil {
		return err
	}

	if !ctx.Renderer.IsText() {
		summaries := make([]map[string]string, 0, len(templates))
		for _, t := range templates {
			metadata := t.GetMetadata()
			summaries = append(summaries, map[string]string{
				"name":        metadata.GetName(),
				"description": metadata.GetDescription(),
			})
		}
		return ctx.Renderer.Render(summaries)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		if len(templates) == 0 {
			_, err := fmt.Fprintln(stdout, "No templates found.")
			return err
		}

		writer := tabwriter.NewWriter(stdout, 0, 8, 2, ' ', 0)
		_, _ = fmt.Fprintln(writer, "NAME\tDESCRIPTION")
		for _, t := range templates {
			metadata := t.GetMetadata()
			_, _ = fmt.Fprintf(writer, "%s\t%s\n", metadata.GetName(), metadata.GetDescription())
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

// normalizeTemplateName converts a template name to a comparable form,
// so both "Health Check Monitor" and "health-check-monitor" match.
func normalizeTemplateName(name string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(name), " ", "-"))
}

// templateResourceFromCanvas creates a clean canvas resource suitable for
// user output, stripping server-only fields like id, organizationId, and isTemplate.
func templateResourceFromCanvas(canvas openapi_client.CanvasesCanvas) models.Canvas {
	var cleanMetadata *openapi_client.CanvasesCanvasMetadata
	if canvas.Metadata != nil {
		cleanMetadata = &openapi_client.CanvasesCanvasMetadata{}
		name := canvas.Metadata.GetName()
		cleanMetadata.SetName(name)
		description := canvas.Metadata.GetDescription()
		if description != "" {
			cleanMetadata.SetDescription(description)
		}
	}

	return models.Canvas{
		APIVersion: core.APIVersion,
		Kind:       models.CanvasKind,
		Metadata:   cleanMetadata,
		Spec:       canvas.Spec,
	}
}

func fetchTemplates(ctx core.CommandContext) ([]openapi_client.CanvasesCanvas, error) {
	response, _, err := ctx.API.CanvasAPI.
		CanvasesListCanvases(ctx.Context).
		IncludeTemplates(true).
		Execute()
	if err != nil {
		return nil, err
	}

	var templates []openapi_client.CanvasesCanvas
	for _, canvas := range response.GetCanvases() {
		if canvas.Metadata != nil && canvas.Metadata.GetIsTemplate() {
			templates = append(templates, canvas)
		}
	}

	return templates, nil
}
