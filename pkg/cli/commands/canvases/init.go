package canvases

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/superplanehq/superplane/pkg/cli/core"
	canvastemplates "github.com/superplanehq/superplane/templates/canvases"
	"gopkg.in/yaml.v3"
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
	filename := name + ".yaml"
	data, err := canvastemplates.FS.ReadFile(filename)
	if err != nil {
		available, listErr := listTemplateNames()
		if listErr != nil {
			return fmt.Errorf("template %q not found", name)
		}
		return fmt.Errorf("template %q not found; available templates: %s", name, strings.Join(available, ", "))
	}

	return c.output(ctx, data)
}

func (c *initCommand) executeListTemplates(ctx core.CommandContext) error {
	templates, err := loadTemplateEntries()
	if err != nil {
		return err
	}

	if !ctx.Renderer.IsText() {
		summaries := make([]map[string]string, 0, len(templates))
		for _, t := range templates {
			summaries = append(summaries, map[string]string{
				"name":        t.key,
				"title":       t.name,
				"description": t.description,
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
			_, _ = fmt.Fprintf(writer, "%s\t%s\n", t.key, t.description)
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

type templateEntry struct {
	key         string
	name        string
	description string
}

func listTemplateNames() ([]string, error) {
	entries, err := loadTemplateEntries()
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.key)
	}
	return names, nil
}

func loadTemplateEntries() ([]templateEntry, error) {
	entries, err := fs.ReadDir(canvastemplates.FS, ".")
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded templates: %w", err)
	}

	var templates []templateEntry
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}

		data, err := canvastemplates.FS.ReadFile(entry.Name())
		if err != nil {
			continue
		}

		meta := parseTemplateMetadata(data)
		key := strings.TrimSuffix(entry.Name(), ".yaml")
		templates = append(templates, templateEntry{
			key:         key,
			name:        meta.name,
			description: meta.description,
		})
	}

	return templates, nil
}

type templateMetadata struct {
	name        string
	description string
}

func parseTemplateMetadata(data []byte) templateMetadata {
	var raw struct {
		Metadata struct {
			Name        string `yaml:"name"`
			Description string `yaml:"description"`
		} `yaml:"metadata"`
	}

	if err := yaml.Unmarshal(data, &raw); err != nil {
		return templateMetadata{}
	}

	return templateMetadata{
		name:        raw.Metadata.Name,
		description: raw.Metadata.Description,
	}
}
