package main

import (
	"bytes"
	"fmt"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

const docsRoot = "docs/integrations"
const coreIndexPath = "docs/integrations/Core/index.yaml"
const coreOutputPath = "docs/integrations/Core.mdx"

var camelBoundary = regexp.MustCompile(`([a-z0-9])([A-Z])`)

type docEntry struct {
	Title         string `yaml:"title"`
	Subtitle      string `yaml:"subtitle"`
	Description   string `yaml:"description"`
	ExampleOutput string `yaml:"exampleOutput"`
}

type docIndex struct {
	Title      string     `yaml:"title"`
	Overview   string     `yaml:"overview"`
	Components []docEntry `yaml:"components"`
	Triggers   []docEntry `yaml:"triggers"`
}

func main() {
	if err := os.MkdirAll(docsRoot, 0o755); err != nil {
		exitWithError(err)
	}

	if err := writeCoreDocsFromIndex(coreIndexPath, coreOutputPath); err != nil {
		exitWithError(err)
	}
}

func writeCoreDocsFromIndex(indexPath string, outputPath string) error {
	indexData, err := os.ReadFile(indexPath)
	if err != nil {
		return err
	}

	var index docIndex
	if err := yaml.Unmarshal(indexData, &index); err != nil {
		return err
	}

	var buf bytes.Buffer
	writeCoreFrontMatter(&buf, strings.TrimSpace(index.Title))
	writeOverviewSection(&buf, index.Overview)
	writeCardGridTriggers(&buf, index.Triggers)
	writeCardGridComponents(&buf, index.Components)
	writeTriggerSection(&buf, index.Triggers)
	writeComponentSection(&buf, index.Components)

	return writeFile(outputPath, buf.Bytes())
}

func writeCoreFrontMatter(buf *bytes.Buffer, title string) {
	if title == "" {
		title = "Core"
	}
	buf.WriteString("---\n")
	buf.WriteString(fmt.Sprintf("title: \"%s\"\n", escapeQuotes(title)))
	buf.WriteString("sidebar:\n")
	buf.WriteString(fmt.Sprintf("  label: \"%s\"\n", escapeQuotes(title)))
	buf.WriteString(fmt.Sprintf("type: \"%s\"\n", escapeQuotes("core")))
	buf.WriteString("---\n\n")
}

func writeComponentSection(buf *bytes.Buffer, components []docEntry) error {
	if len(components) == 0 {
		return nil
	}

	for _, component := range components {
		buf.WriteString(fmt.Sprintf("<a id=\"%s\"></a>\n\n", slugify(component.Title)))
		buf.WriteString(fmt.Sprintf("## %s\n\n", component.Title))
		writeParagraph(buf, component.Description)
		if err := writeExampleSection("Example Output", component.ExampleOutput, buf); err != nil {
			return err
		}
	}

	return nil
}

func writeTriggerSection(buf *bytes.Buffer, triggers []docEntry) error {
	if len(triggers) == 0 {
		return nil
	}

	for _, trigger := range triggers {
		buf.WriteString(fmt.Sprintf("<a id=\"%s\"></a>\n\n", slugify(trigger.Title)))
		buf.WriteString(fmt.Sprintf("## %s\n\n", trigger.Title))
		writeParagraph(buf, trigger.Description)
		if err := writeExampleSection("Example Data", trigger.ExampleOutput, buf); err != nil {
			return err
		}
	}

	return nil
}

func writeCardGridComponents(buf *bytes.Buffer, components []docEntry) {
	if len(components) == 0 {
		return
	}

	writeCardGridImport(buf)
	buf.WriteString("## Components\n\n")
	buf.WriteString("<CardGrid>\n")
	for _, component := range components {
		description := strings.TrimSpace(component.Subtitle)
		if description == "" {
			description = strings.TrimSpace(component.Description)
		}
		buf.WriteString(fmt.Sprintf("  <LinkCard title=\"%s\" href=\"#%s\" description=\"%s\" />\n",
			escapeQuotes(component.Title),
			slugify(component.Title),
			escapeQuotes(description),
		))
	}
	buf.WriteString("</CardGrid>\n\n")
}

func writeCardGridTriggers(buf *bytes.Buffer, triggers []docEntry) {
	if len(triggers) == 0 {
		return
	}

	writeCardGridImport(buf)
	buf.WriteString("## Triggers\n\n")
	buf.WriteString("<CardGrid>\n")
	for _, trigger := range triggers {
		description := strings.TrimSpace(trigger.Subtitle)
		if description == "" {
			description = strings.TrimSpace(trigger.Description)
		}
		buf.WriteString(fmt.Sprintf("  <LinkCard title=\"%s\" href=\"#%s\" description=\"%s\" />\n",
			escapeQuotes(trigger.Title),
			slugify(trigger.Title),
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

func writeOverviewSection(buf *bytes.Buffer, description string) {
	trimmed := strings.TrimSpace(description)
	if trimmed == "" {
		return
	}
	buf.WriteString(trimmed)
	buf.WriteString("\n\n")
}

func writeExampleSection(title string, examplePath string, buf *bytes.Buffer) error {
	trimmed := strings.TrimSpace(examplePath)
	if trimmed == "" {
		return nil
	}

	raw, err := os.ReadFile(trimmed)
	if err != nil {
		return err
	}
	trimmedRaw := strings.TrimSpace(string(raw))
	if trimmedRaw == "" {
		return nil
	}
	buf.WriteString(fmt.Sprintf("### %s\n\n", title))
	buf.WriteString("```json\n")
	buf.WriteString(trimmedRaw)
	buf.WriteString("\n```\n\n")
	return nil
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

func writeCardGridImport(buf *bytes.Buffer) {
	content := buf.String()
	if strings.Contains(content, "CardGrid") {
		return
	}
	buf.WriteString("import { CardGrid, LinkCard } from \"@astrojs/starlight/components\";\n\n")
}

func escapeQuotes(value string) string {
	return strings.ReplaceAll(value, "\"", "\\\"")
}

func exitWithError(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
