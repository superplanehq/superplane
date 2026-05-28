package docs

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
	"github.com/superplanehq/superplane/pkg/registryimports"
)

var _ = registryimports.Loaded

type File struct {
	Name    string
	Content []byte
}

var (
	camelBoundary = regexp.MustCompile(`([a-z0-9])([A-Z])`)
	htmlTagRe     = regexp.MustCompile(`<([a-zA-Z/][^>]*)>`)
)

func GenerateFiles() ([]File, error) {
	reg, err := registry.NewRegistry(crypto.NewNoOpEncryptor(), registry.HTTPOptions{})
	if err != nil {
		return nil, err
	}

	integrations := reg.ListIntegrations()
	sort.Slice(integrations, func(i, j int) bool {
		return integrations[i].Label() < integrations[j].Label()
	})

	files := make([]File, 0, len(integrations)+1)

	coreDoc, err := renderCoreComponentsDoc(reg.ListActions(), reg.ListTriggers())
	if err != nil {
		return nil, err
	}
	if len(coreDoc) > 0 {
		files = append(files, File{Name: "Core.mdx", Content: coreDoc})
	}

	for _, integration := range integrations {
		doc, err := renderIntegrationDoc(integration)
		if err != nil {
			return nil, err
		}
		files = append(files, File{
			Name:    fmt.Sprintf("%s.mdx", integrationFilename(integration)),
			Content: doc,
		})
	}

	return files, nil
}

func GenerateComponentIndexFile() (File, error) {
	reg, err := registry.NewRegistry(crypto.NewNoOpEncryptor(), registry.HTTPOptions{})
	if err != nil {
		return File{}, err
	}

	integrations := reg.ListIntegrations()
	sort.Slice(integrations, func(i, j int) bool {
		return integrations[i].Label() < integrations[j].Label()
	})

	content := renderComponentIndex(reg.ListActions(), reg.ListTriggers(), integrations)
	return File{Name: "Index.md", Content: content}, nil
}

func WriteFiles(outputDir string) error {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return err
	}

	files, err := GenerateFiles()
	if err != nil {
		return err
	}

	for _, file := range files {
		if err := os.WriteFile(filepath.Join(outputDir, file.Name), file.Content, 0o644); err != nil {
			return err
		}
	}

	return nil
}

func renderIntegrationDoc(integration core.Integration) ([]byte, error) {
	actions := integration.Actions()
	triggers := integration.Triggers()

	sort.Slice(actions, func(i, j int) bool { return actions[i].Name() < actions[j].Name() })
	sort.Slice(triggers, func(i, j int) bool { return triggers[i].Name() < triggers[j].Name() })

	var buf bytes.Buffer
	writeFrontMatter(&buf, integration.Label(), nil)
	writeOverviewSection(&buf, sanitizeHTMLTags(integration.Description()))
	writeCardGridImport(&buf, triggers, actions)
	writeCardGridTriggers(&buf, triggers)
	writeCardGridActions(&buf, actions)

	if instructions := strings.TrimSpace(integration.Instructions()); instructions != "" {
		buf.WriteString("## Instructions\n\n")
		buf.WriteString(sanitizeHTMLTags(instructions))
		buf.WriteString("\n\n")
	}

	writeTriggerSection(&buf, triggers)
	writeActionSection(&buf, actions)
	return buf.Bytes(), nil
}

func renderCoreComponentsDoc(actions []core.Action, triggers []core.Trigger) ([]byte, error) {
	if len(actions) == 0 && len(triggers) == 0 {
		return nil, nil
	}

	sort.Slice(actions, func(i, j int) bool { return actions[i].Name() < actions[j].Name() })
	sort.Slice(triggers, func(i, j int) bool { return triggers[i].Name() < triggers[j].Name() })

	var buf bytes.Buffer
	coreOrder := 1
	writeFrontMatter(&buf, "Core", &coreOrder)
	writeOverviewSection(&buf, "Built-in SuperPlane components.")
	writeCardGridImport(&buf, triggers, actions)
	writeCardGridTriggers(&buf, triggers)
	writeCardGridActions(&buf, actions)
	writeTriggerSection(&buf, triggers)
	writeActionSection(&buf, actions)
	return buf.Bytes(), nil
}

func renderComponentIndex(actions []core.Action, triggers []core.Trigger, integrations []core.Integration) []byte {
	sort.Slice(actions, func(i, j int) bool { return actions[i].Name() < actions[j].Name() })
	sort.Slice(triggers, func(i, j int) bool { return triggers[i].Name() < triggers[j].Name() })

	var buf bytes.Buffer
	buf.WriteString("# SuperPlane Component Index\n\n")
	buf.WriteString("Fast lookup for canvas YAML keys. Use this before searching full component docs.\n\n")

	writeIndexEntries(&buf, "Core Triggers", "Trigger", triggerIndexEntries(triggers))
	writeIndexEntries(&buf, "Core Actions", "Component", actionIndexEntries(actions))

	for _, integration := range integrations {
		integrationTriggers := integration.Triggers()
		integrationActions := integration.Actions()
		sort.Slice(integrationTriggers, func(i, j int) bool { return integrationTriggers[i].Name() < integrationTriggers[j].Name() })
		sort.Slice(integrationActions, func(i, j int) bool { return integrationActions[i].Name() < integrationActions[j].Name() })

		writeIndexEntries(
			&buf,
			fmt.Sprintf("%s Triggers", integration.Label()),
			"Trigger",
			triggerIndexEntries(integrationTriggers),
		)
		writeIndexEntries(
			&buf,
			fmt.Sprintf("%s Actions", integration.Label()),
			"Component",
			actionIndexEntries(integrationActions),
		)
	}

	return buf.Bytes()
}

type componentIndexEntry struct {
	Key         string
	Label       string
	Description string
}

func actionIndexEntries(actions []core.Action) []componentIndexEntry {
	entries := make([]componentIndexEntry, 0, len(actions))
	for _, action := range actions {
		entries = append(entries, componentIndexEntry{
			Key:         action.Name(),
			Label:       action.Label(),
			Description: action.Description(),
		})
	}
	return entries
}

func triggerIndexEntries(triggers []core.Trigger) []componentIndexEntry {
	entries := make([]componentIndexEntry, 0, len(triggers))
	for _, trigger := range triggers {
		entries = append(entries, componentIndexEntry{
			Key:         trigger.Name(),
			Label:       trigger.Label(),
			Description: trigger.Description(),
		})
	}
	return entries
}

func writeIndexEntries(buf *bytes.Buffer, title, keyLabel string, entries []componentIndexEntry) {
	if len(entries) == 0 {
		return
	}

	buf.WriteString(fmt.Sprintf("## %s\n\n", title))
	buf.WriteString(fmt.Sprintf("| Label | %s key | Description |\n", keyLabel))
	buf.WriteString("| --- | --- | --- |\n")
	for _, entry := range entries {
		buf.WriteString(fmt.Sprintf(
			"| %s | `%s` | %s |\n",
			escapeTableCell(entry.Label),
			entry.Key,
			escapeTableCell(entry.Description),
		))
	}
	buf.WriteString("\n")
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

func writeActionSection(buf *bytes.Buffer, actions []core.Action) {
	if len(actions) == 0 {
		return
	}

	for _, action := range actions {
		buf.WriteString(fmt.Sprintf("<a id=\"%s\"></a>\n\n", slugify(action.Label())))
		buf.WriteString(fmt.Sprintf("## %s\n\n", action.Label()))
		writeKeyLine(buf, "Component key", action.Name())

		doc := action.Documentation()
		if doc != "" {
			writeParagraph(buf, adjustHeadingLevels(doc))
		} else {
			writeParagraph(buf, action.Description())
		}

		writeExampleSection("Example Output", action.ExampleOutput(), buf)
	}
}

func writeTriggerSection(buf *bytes.Buffer, triggers []core.Trigger) {
	if len(triggers) == 0 {
		return
	}

	for _, trigger := range triggers {
		buf.WriteString(fmt.Sprintf("<a id=\"%s\"></a>\n\n", slugify(trigger.Label())))
		buf.WriteString(fmt.Sprintf("## %s\n\n", trigger.Label()))
		writeKeyLine(buf, "Trigger key", trigger.Name())

		doc := trigger.Documentation()
		if doc != "" {
			writeParagraph(buf, adjustHeadingLevels(doc))
		} else {
			writeParagraph(buf, trigger.Description())
		}

		writeExampleSection("Example Data", trigger.ExampleData(), buf)
	}
}

func writeKeyLine(buf *bytes.Buffer, label, key string) {
	trimmed := strings.TrimSpace(key)
	if trimmed == "" {
		return
	}
	buf.WriteString(fmt.Sprintf("**%s:** `%s`\n\n", label, trimmed))
}

func writeCardGridImport(buf *bytes.Buffer, triggers []core.Trigger, actions []core.Action) {
	if len(triggers) == 0 && len(actions) == 0 {
		return
	}

	buf.WriteString("import { CardGrid, LinkCard } from \"@astrojs/starlight/components\";\n\n")
}

func writeCardGridActions(buf *bytes.Buffer, actions []core.Action) {
	if len(actions) == 0 {
		return
	}

	buf.WriteString("## Actions\n\n")
	buf.WriteString("<CardGrid>\n")
	for _, action := range actions {
		description := strings.TrimSpace(action.Description())
		buf.WriteString(fmt.Sprintf("  <LinkCard title=\"%s\" href=\"#%s\" description=\"%s\" />\n",
			escapeQuotes(action.Label()),
			slugify(action.Label()),
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
	buf.WriteString(sanitizeHTMLTags(trimmed))
	buf.WriteString("\n\n")
}

func adjustHeadingLevels(text string) string {
	lines := strings.Split(text, "\n")
	var result []string

	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			trimmed := strings.TrimSpace(line)
			level := 0
			for _, r := range trimmed {
				if r == '#' {
					level++
				} else {
					break
				}
			}
			if level > 0 && level < 6 {
				result = append(result, strings.Repeat("#", level+1)+trimmed[level:])
			} else {
				result = append(result, line)
			}
			continue
		}

		result = append(result, line)
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

func sanitizeHTMLTags(content string) string {
	lines := strings.Split(content, "\n")
	inCodeBlock := false
	var result []string

	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			inCodeBlock = !inCodeBlock
			result = append(result, line)
			continue
		}

		if inCodeBlock {
			result = append(result, line)
			continue
		}

		parts := strings.Split(line, "`")
		for i := range parts {
			if i%2 == 0 {
				parts[i] = htmlTagRe.ReplaceAllString(parts[i], "&lt;$1&gt;")
				parts[i] = strings.ReplaceAll(parts[i], "{", "&lbrace;")
				parts[i] = strings.ReplaceAll(parts[i], "}", "&rbrace;")
			}
		}
		result = append(result, strings.Join(parts, "`"))
	}

	return strings.Join(result, "\n")
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

func escapeTableCell(value string) string {
	value = strings.ReplaceAll(strings.TrimSpace(value), "\n", " ")
	return strings.ReplaceAll(value, "|", "\\|")
}
