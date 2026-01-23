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

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/registry"

	// Import integrations to register them via init().
	_ "github.com/superplanehq/superplane/pkg/applications/dash0"
	_ "github.com/superplanehq/superplane/pkg/applications/github"
	_ "github.com/superplanehq/superplane/pkg/applications/openai"
	_ "github.com/superplanehq/superplane/pkg/applications/pagerduty"
	_ "github.com/superplanehq/superplane/pkg/applications/semaphore"
	_ "github.com/superplanehq/superplane/pkg/applications/slack"
	_ "github.com/superplanehq/superplane/pkg/applications/smtp"
	_ "github.com/superplanehq/superplane/pkg/components/approval"
	_ "github.com/superplanehq/superplane/pkg/components/filter"
	_ "github.com/superplanehq/superplane/pkg/components/http"
	_ "github.com/superplanehq/superplane/pkg/components/if"
	_ "github.com/superplanehq/superplane/pkg/components/merge"
	_ "github.com/superplanehq/superplane/pkg/components/noop"
	_ "github.com/superplanehq/superplane/pkg/components/timegate"
	_ "github.com/superplanehq/superplane/pkg/components/wait"
	_ "github.com/superplanehq/superplane/pkg/triggers/schedule"
	_ "github.com/superplanehq/superplane/pkg/triggers/start"
	_ "github.com/superplanehq/superplane/pkg/triggers/webhook"
)

const docsRoot = "docs/integrations"

var camelBoundary = regexp.MustCompile(`([a-z0-9])([A-Z])`)

func main() {
	reg := registry.NewRegistry(crypto.NewNoOpEncryptor())
	apps := reg.ListApplications()

	if err := os.MkdirAll(docsRoot, 0o755); err != nil {
		exitWithError(err)
	}

	if err := writeCoreComponentsDoc(reg.ListComponents(), reg.ListTriggers()); err != nil {
		exitWithError(err)
	}

	for _, app := range apps {
		if err := writeAppDocs(app); err != nil {
			exitWithError(err)
		}
	}
}

func writeAppDocs(app core.Application) error {
	if err := os.MkdirAll(docsRoot, 0o755); err != nil {
		return err
	}

	components := app.Components()
	triggers := app.Triggers()

	sort.Slice(components, func(i, j int) bool { return components[i].Name() < components[j].Name() })
	sort.Slice(triggers, func(i, j int) bool { return triggers[i].Name() < triggers[j].Name() })

	return writeAppIndex(filepath.Join(docsRoot, fmt.Sprintf("%s.mdx", appFilename(app))), app, components, triggers)
}

func writeCoreComponentsDoc(components []core.Component, triggers []core.Trigger) error {
	if len(components) == 0 {
		if len(triggers) == 0 {
			return nil
		}
	}

	sort.Slice(components, func(i, j int) bool { return components[i].Name() < components[j].Name() })
	sort.Slice(triggers, func(i, j int) bool { return triggers[i].Name() < triggers[j].Name() })

	var buf bytes.Buffer
	writeCoreFrontMatter(&buf)
	writeOverviewSection(&buf, "Built-in SuperPlane components.")
	writeCardGridComponents(&buf, components)
	writeCardGridTriggers(&buf, triggers)
	writeComponentSection(&buf, components)
	writeTriggerSection(&buf, triggers)

	return writeFile(filepath.Join(docsRoot, "Core.mdx"), buf.Bytes())
}

func writeAppIndex(
	path string,
	app core.Application,
	components []core.Component,
	triggers []core.Trigger,
) error {
	var buf bytes.Buffer
	writeAppFrontMatter(&buf, app)

	writeOverviewSection(&buf, app.Description())
	writeCardGridComponents(&buf, components)
	writeCardGridTriggers(&buf, triggers)

	if instructions := strings.TrimSpace(app.InstallationInstructions()); instructions != "" {
		buf.WriteString("## Installation\n\n")
		buf.WriteString(instructions)
		buf.WriteString("\n\n")
	}

	writeComponentSection(&buf, components)
	writeTriggerSection(&buf, triggers)

	return writeFile(path, buf.Bytes())
}

func writeAppFrontMatter(buf *bytes.Buffer, app core.Application) {
	buf.WriteString("---\n")
	buf.WriteString(fmt.Sprintf("title: \"%s\"\n", escapeQuotes(app.Label())))
	buf.WriteString("sidebar:\n")
	buf.WriteString(fmt.Sprintf("  label: \"%s\"\n", escapeQuotes(app.Label())))
	buf.WriteString(fmt.Sprintf("type: \"%s\"\n", escapeQuotes("application")))
	buf.WriteString(fmt.Sprintf("name: \"%s\"\n", escapeQuotes(app.Name())))
	buf.WriteString(fmt.Sprintf("label: \"%s\"\n", escapeQuotes(app.Label())))
	buf.WriteString("---\n\n")
}

func writeCoreFrontMatter(buf *bytes.Buffer) {
	buf.WriteString("---\n")
	buf.WriteString(fmt.Sprintf("title: \"%s\"\n", escapeQuotes("Core")))
	buf.WriteString("sidebar:\n")
	buf.WriteString(fmt.Sprintf("  label: \"%s\"\n", escapeQuotes("Core")))
	buf.WriteString(fmt.Sprintf("type: \"%s\"\n", escapeQuotes("core")))
	buf.WriteString("---\n\n")
}

func writeComponentSection(buf *bytes.Buffer, components []core.Component) {
	if len(components) == 0 {
		return
	}

	for _, component := range components {
		buf.WriteString(fmt.Sprintf("<a id=\"%s\"></a>\n\n", slugify(component.Label())))
		buf.WriteString(fmt.Sprintf("## %s\n\n", component.Label()))
		writeParagraph(buf, component.Description())
		writeConfigurationSection(buf, component.Configuration())
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
		writeParagraph(buf, trigger.Description())
		config := actions.AppendGlobalTriggerFields(trigger.Configuration())
		writeConfigurationSection(buf, config)
		writeExampleSection("Example Data", trigger.ExampleData(), buf)
	}
}

func writeCardGridComponents(buf *bytes.Buffer, components []core.Component) {
	if len(components) == 0 {
		return
	}

	buf.WriteString("import { CardGrid, LinkCard } from \"@astrojs/starlight/components\";\n\n")
	buf.WriteString("## Components\n\n")
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

func writeOverviewSection(buf *bytes.Buffer, description string) {
	trimmed := strings.TrimSpace(description)
	if trimmed == "" {
		return
	}
	buf.WriteString("## Overview\n\n")
	buf.WriteString(trimmed)
	buf.WriteString("\n\n")
}

func writeOutputChannels(buf *bytes.Buffer, channels []core.OutputChannel) {
	if len(channels) == 0 {
		channels = []core.OutputChannel{core.DefaultOutputChannel}
	}

	buf.WriteString("## Output Channels\n\n")
	buf.WriteString("| Name | Label | Description |\n")
	buf.WriteString("| --- | --- | --- |\n")
	for _, channel := range channels {
		buf.WriteString(fmt.Sprintf("| %s | %s | %s |\n",
			formatTableValue(channel.Name),
			formatTableValue(channel.Label),
			formatTableValue(channel.Description),
		))
	}
	buf.WriteString("\n")
}

func writeConfigurationSection(buf *bytes.Buffer, fields []configuration.Field) {
	if len(fields) == 0 {
		return
	}

	buf.WriteString("### Configuration\n\n")
	buf.WriteString("| Name | Label | Type | Required | Description |\n")
	buf.WriteString("| --- | --- | --- | --- | --- |\n")
	for _, field := range fields {
		required := "no"
		if field.Required {
			required = "yes"
		}
		buf.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s |\n",
			formatTableValue(field.Name),
			formatTableValue(field.Label),
			formatTableValue(field.Type),
			required,
			formatTableValue(field.Description),
		))
	}
	buf.WriteString("\n")
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

func appFilename(app core.Application) string {
	label := strings.TrimSpace(app.Label())
	if label == "" {
		return slugify(app.Name())
	}
	return label
}

func formatTableValue(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "-"
	}
	escaped := strings.ReplaceAll(trimmed, "|", "\\|")
	escaped = strings.ReplaceAll(escaped, "{", "&#123;")
	escaped = strings.ReplaceAll(escaped, "}", "&#125;")
	return escaped
}

func escapeQuotes(value string) string {
	return strings.ReplaceAll(value, "\"", "\\\"")
}

func exitWithError(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
