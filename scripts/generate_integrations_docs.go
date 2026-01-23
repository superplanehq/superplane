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
)

const docsRoot = "docs/integrations"

var camelBoundary = regexp.MustCompile(`([a-z0-9])([A-Z])`)

func main() {
	reg := registry.NewRegistry(crypto.NewNoOpEncryptor())
	apps := reg.ListApplications()

	if err := os.MkdirAll(docsRoot, 0o755); err != nil {
		exitWithError(err)
	}

	for _, app := range apps {
		if err := writeAppDocs(app); err != nil {
			exitWithError(err)
		}
	}
}

func writeAppDocs(app core.Application) error {
	appSlug := slugify(app.Name())
	baseDir := filepath.Join(docsRoot, appSlug)
	componentDir := filepath.Join(baseDir, "components")
	triggerDir := filepath.Join(baseDir, "triggers")

	if err := os.MkdirAll(componentDir, 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(triggerDir, 0o755); err != nil {
		return err
	}

	components := app.Components()
	triggers := app.Triggers()

	sort.Slice(components, func(i, j int) bool { return components[i].Name() < components[j].Name() })
	sort.Slice(triggers, func(i, j int) bool { return triggers[i].Name() < triggers[j].Name() })

	for _, component := range components {
		if err := writeComponentDoc(componentDir, app, component); err != nil {
			return err
		}
	}

	for _, trigger := range triggers {
		if err := writeTriggerDoc(triggerDir, app, trigger); err != nil {
			return err
		}
	}

	return writeAppIndex(baseDir, app, components, triggers)
}

func writeAppIndex(dir string, app core.Application, components []core.Component, triggers []core.Trigger) error {
	var buf bytes.Buffer
	writeFrontMatter(&buf, map[string]string{
		"type":  "application",
		"name":  app.Name(),
		"label": app.Label(),
	})

	buf.WriteString(fmt.Sprintf("# %s\n\n", app.Label()))
	writeParagraph(&buf, app.Description())

	if instructions := strings.TrimSpace(app.InstallationInstructions()); instructions != "" {
		buf.WriteString("## Installation\n\n")
		buf.WriteString(instructions)
		buf.WriteString("\n\n")
	}

	if len(components) > 0 {
		buf.WriteString("## Components\n\n")
		for _, component := range components {
			name := component.Name()
			label := component.Label()
			link := fmt.Sprintf("components/%s.md", slugify(componentNameOnly(name)))
			buf.WriteString(fmt.Sprintf("- [%s](%s)\n", label, link))
		}
		buf.WriteString("\n")
	}

	if len(triggers) > 0 {
		buf.WriteString("## Triggers\n\n")
		for _, trigger := range triggers {
			name := trigger.Name()
			label := trigger.Label()
			link := fmt.Sprintf("triggers/%s.md", slugify(componentNameOnly(name)))
			buf.WriteString(fmt.Sprintf("- [%s](%s)\n", label, link))
		}
		buf.WriteString("\n")
	}

	return writeFile(filepath.Join(dir, "index.md"), buf.Bytes())
}

func writeComponentDoc(dir string, app core.Application, component core.Component) error {
	var buf bytes.Buffer
	writeFrontMatter(&buf, map[string]string{
		"type":  "component",
		"app":   app.Name(),
		"name":  component.Name(),
		"label": component.Label(),
	})

	buf.WriteString(fmt.Sprintf("# %s\n\n", component.Label()))
	writeParagraph(&buf, component.Description())

	writeOutputChannels(&buf, component.OutputChannels(nil))
	writeConfiguration(&buf, component.Configuration())
	writeExample("Example Output", component.ExampleOutput(), &buf)

	filename := filepath.Join(dir, fmt.Sprintf("%s.md", slugify(componentNameOnly(component.Name()))))
	return writeFile(filename, buf.Bytes())
}

func writeTriggerDoc(dir string, app core.Application, trigger core.Trigger) error {
	var buf bytes.Buffer
	writeFrontMatter(&buf, map[string]string{
		"type":  "trigger",
		"app":   app.Name(),
		"name":  trigger.Name(),
		"label": trigger.Label(),
	})

	buf.WriteString(fmt.Sprintf("# %s\n\n", trigger.Label()))
	writeParagraph(&buf, trigger.Description())

	config := actions.AppendGlobalTriggerFields(trigger.Configuration())
	writeConfiguration(&buf, config)
	writeExample("Example Data", trigger.ExampleData(), &buf)

	filename := filepath.Join(dir, fmt.Sprintf("%s.md", slugify(componentNameOnly(trigger.Name()))))
	return writeFile(filename, buf.Bytes())
}

func writeFrontMatter(buf *bytes.Buffer, fields map[string]string) {
	buf.WriteString("---\n")
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		value := strings.TrimSpace(fields[key])
		buf.WriteString(fmt.Sprintf("%s: \"%s\"\n", key, escapeQuotes(value)))
	}
	buf.WriteString("---\n\n")
}

func writeParagraph(buf *bytes.Buffer, text string) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return
	}
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

func writeConfiguration(buf *bytes.Buffer, fields []configuration.Field) {
	if len(fields) == 0 {
		return
	}

	buf.WriteString("## Configuration\n\n")
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

func writeExample(title string, data map[string]any, buf *bytes.Buffer) {
	if len(data) == 0 {
		return
	}
	raw, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return
	}
	buf.WriteString(fmt.Sprintf("## %s\n\n", title))
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

func componentNameOnly(value string) string {
	parts := strings.SplitN(value, ".", 2)
	if len(parts) == 2 {
		return parts[1]
	}
	return value
}

func formatTableValue(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "-"
	}
	escaped := strings.ReplaceAll(trimmed, "|", "\\|")
	return escaped
}

func escapeQuotes(value string) string {
	return strings.ReplaceAll(value, "\"", "\\\"")
}

func exitWithError(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
