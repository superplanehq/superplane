package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/registry"

	// Import server to auto-register all integrations, components, and triggers via init()
	_ "github.com/superplanehq/superplane/pkg/server"
)

const docsRoot = "docs/components"

var camelBoundary = regexp.MustCompile(`([a-z0-9])([A-Z])`)

func main() {
	createOutputDirectory()

	reg, err := registry.NewRegistry(crypto.NewNoOpEncryptor(), registry.HTTPOptions{})
	if err != nil {
		exitWithError(err)
	}

	integrations := reg.ListIntegrations()

	// Sort integrations by name
	sort.Slice(integrations, func(i, j int) bool {
		return integrations[i].Label() < integrations[j].Label()
	})

	if err := writeCoreComponentsDoc(reg.ListComponents(), reg.ListTriggers()); err != nil {
		exitWithError(err)
	}

	for _, integration := range integrations {
		if err := writeIntegrationDocs(integration); err != nil {
			exitWithError(err)
		}
	}
}

func createOutputDirectory() {
	if err := os.MkdirAll(docsRoot, 0o755); err != nil {
		os.Exit(1)
	}
}

func writeIntegrationDocs(integration core.Integration) error {
	components := integration.Components()
	triggers := integration.Triggers()

	sort.Slice(components, func(i, j int) bool { return components[i].Name() < components[j].Name() })
	sort.Slice(triggers, func(i, j int) bool { return triggers[i].Name() < triggers[j].Name() })

	return writeIntegrationIndex(filepath.Join(docsRoot, fmt.Sprintf("%s.mdx", integrationFilename(integration))), integration, components, triggers)
}

func writeCoreComponentsDoc(components []core.Component, triggers []core.Trigger) error {
	if len(components) == 0 && len(triggers) == 0 {
		return nil
	}

	sort.Slice(components, func(i, j int) bool { return components[i].Name() < components[j].Name() })
	sort.Slice(triggers, func(i, j int) bool { return triggers[i].Name() < triggers[j].Name() })

	var buf bytes.Buffer
	coreOrder := 1
	writeFrontMatter(&buf, "Core", &coreOrder)
	writeOverviewSection(&buf, "Built-in SuperPlane components.")
	writeCardGridTriggers(&buf, triggers)
	writeCardGridComponents(&buf, components)
	writeTriggerSection(&buf, triggers)
	writeComponentSection(&buf, components)

	return writeFile(filepath.Join(docsRoot, "Core.mdx"), buf.Bytes())
}

func writeIntegrationIndex(
	path string,
	integration core.Integration,
	components []core.Component,
	triggers []core.Trigger,
) error {
	var buf bytes.Buffer
	writeFrontMatter(&buf, integration.Label(), nil)

	writeOverviewSection(&buf, integration.Description())
	writeCardGridTriggers(&buf, triggers)
	writeCardGridComponents(&buf, components)

	if instructions := strings.TrimSpace(integration.Instructions()); instructions != "" {
		buf.WriteString("## Instructions\n\n")
		buf.WriteString(instructions)
		buf.WriteString("\n\n")
	}

	writeTriggerSection(&buf, triggers)
	writeComponentSection(&buf, components)

	return writeFile(path, buf.Bytes())
}

func writeFrontMatter(buf *bytes.Buffer, title string, order *int) {
	buf.WriteString("---\n")
	buf.WriteString(fmt.Sprintf("title: \"%s\"\n", escapeQuotes(title)))
	if order != nil {
		buf.WriteString("sidebar:\n")
		buf.WriteString(fmt.Sprintf("  order: %d\n", *order))
	}
	buf.WriteString("---\n\n")
}

func writeComponentSection(buf *bytes.Buffer, components []core.Component) {
	if len(components) == 0 {
		return
	}

	for _, component := range components {
		buf.WriteString(fmt.Sprintf("<a id=\"%s\"></a>\n\n", slugify(component.Label())))
		buf.WriteString(fmt.Sprintf("## %s\n\n", component.Label()))

		// Write documentation if available, otherwise fall back to description
		doc := component.Documentation()
		if doc != "" {
			adjustedDoc := adjustHeadingLevels(doc)
			writeParagraph(buf, adjustedDoc)
		} else {
			writeParagraph(buf, component.Description())
		}

		writeExampleSection("Example Output", component.ExampleOutput(), buf)
	}
}

func writeTriggerSection(buf *bytes.Buffer, triggers []core.Trigger) {
	if len(triggers) == 0 {
		return
	}

	for _, trigger := range triggers {
		buf.WriteString(fmt.Sprintf("<a id=\"%s\"></a>\n\n", slugify(trigger.Label())))
		buf.WriteString(fmt.Sprintf("## %s\n\n", trigger.Label()))

		// Write documentation if available, otherwise fall back to description
		doc := trigger.Documentation()
		if doc != "" {
			adjustedDoc := adjustHeadingLevels(doc)
			writeParagraph(buf, adjustedDoc)
		} else {
			writeParagraph(buf, trigger.Description())
		}

		writeExampleSection("Example Data", trigger.ExampleData(), buf)
	}
}

func writeCardGridComponents(buf *bytes.Buffer, components []core.Component) {
	if len(components) == 0 {
		return
	}

	buf.WriteString("import { CardGrid, LinkCard } from \"@astrojs/starlight/components\";\n\n")
	buf.WriteString("## Actions\n\n")
	buf.WriteString("<CardGrid>\n")
	for _, component := range components {
		description := strings.TrimSpace(component.Description())
		buf.WriteString(fmt.Sprintf("  <LinkCard title=\"%s\" href=\"#%s\" description=\"%s\" />\n",
			escapeQuotes(component.Label()),
			slugify(component.Label()),
			escapeQuotes(description),
		))
	}
	buf.WriteString("</CardGrid>\n\n")
}

func writeCardGridTriggers(buf *bytes.Buffer, triggers []core.Trigger) {
	if len(triggers) == 0 {
		return
	}

	buf.WriteString("## Triggers\n\n")
	buf.WriteString("<CardGrid>\n")
	for _, trigger := range triggers {
		description := strings.TrimSpace(trigger.Description())
		buf.WriteString(fmt.Sprintf("  <LinkCard title=\"%s\" href=\"#%s\" description=\"%s\" />\n",
			escapeQuotes(trigger.Label()),
			slugify(trigger.Label()),
			escapeQuotes(description),
		))
	}
	buf.WriteString("</CardGrid>\n\n")
}

func writeParagraph(buf *bytes.Buffer, text string) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return
	}
	buf.WriteString(trimmed)
	buf.WriteString("\n\n")
}

// adjustHeadingLevels increments all markdown heading levels by 1
// H2 (##) becomes H3 (###), H3 becomes H4, etc.
func adjustHeadingLevels(text string) string {
	lines := strings.Split(text, "\n")
	var result []string

	for _, line := range lines {
		// Check if line is a markdown heading (starts with #)
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			// Count leading # characters
			trimmed := strings.TrimSpace(line)
			level := 0
			for _, r := range trimmed {
				if r == '#' {
					level++
				} else {
					break
				}
			}
			// Increment heading level by adding one more #
			if level > 0 && level < 6 {
				// Add one more # to increase the heading level
				result = append(result, strings.Repeat("#", level+1)+trimmed[level:])
			} else {
				result = append(result, line)
			}
		} else {
			result = append(result, line)
		}
	}

	return strings.Join(result, "\n")
}

func writeOverviewSection(buf *bytes.Buffer, description string) {
	buf.WriteString(description)
	buf.WriteString("\n\n")
}

func writeExampleSection(title string, data map[string]any, buf *bytes.Buffer) {
	if len(data) == 0 {
		return
	}
	raw, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return
	}
	buf.WriteString(fmt.Sprintf("### %s\n\n", title))
	buf.WriteString("```json\n")
	buf.Write(raw)
	buf.WriteString("\n```\n\n")
}

func writeFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0o644)
}

func slugify(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "unknown"
	}
	snake := strings.ReplaceAll(trimmed, "_", "-")
	withDashes := camelBoundary.ReplaceAllString(snake, "$1-$2")
	withDashes = strings.ReplaceAll(withDashes, " ", "-")
	withDashes = strings.ReplaceAll(withDashes, ".", "-")
	return strings.ToLower(withDashes)
}

func integrationFilename(integration core.Integration) string {
	label := strings.TrimSpace(integration.Label())
	label = strings.ReplaceAll(label, " ", "")

	if label == "" {
		return slugify(integration.Name())
	}

	return label
}

func escapeQuotes(value string) string {
	return strings.ReplaceAll(value, "\"", "\\\"")
}

func exitWithError(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
