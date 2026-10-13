package docs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
)

type fakeAction struct {
	core.Action
	name          string
	label         string
	description   string
	documentation string
	exampleOutput map[string]any
}

func (f fakeAction) Name() string                  { return f.name }
func (f fakeAction) Label() string                 { return f.label }
func (f fakeAction) Description() string           { return f.description }
func (f fakeAction) Documentation() string         { return f.documentation }
func (f fakeAction) ExampleOutput() map[string]any { return f.exampleOutput }

type fakeTrigger struct {
	core.Trigger
	name          string
	label         string
	description   string
	documentation string
	exampleData   map[string]any
}

func (f fakeTrigger) Name() string                { return f.name }
func (f fakeTrigger) Label() string               { return f.label }
func (f fakeTrigger) Description() string         { return f.description }
func (f fakeTrigger) Documentation() string       { return f.documentation }
func (f fakeTrigger) ExampleData() map[string]any { return f.exampleData }

type fakeIntegration struct {
	core.Integration
	name         string
	label        string
	description  string
	instructions string
	actions      []core.Action
	triggers     []core.Trigger
}

func (f fakeIntegration) Name() string             { return f.name }
func (f fakeIntegration) Label() string            { return f.label }
func (f fakeIntegration) Description() string      { return f.description }
func (f fakeIntegration) Instructions() string     { return f.instructions }
func (f fakeIntegration) Actions() []core.Action   { return f.actions }
func (f fakeIntegration) Triggers() []core.Trigger { return f.triggers }
func TestSlugify(t *testing.T) {
	cases := []struct{ in, want string }{
		{"GitHub Actions", "git-hub-actions"},
		{"getWeatherAPI", "get-weather-api"},
		{"snake_case_name", "snake-case-name"},
		{"v1.2 Release", "v1-2-release"},
		{"", "unknown"},
	}
	for _, c := range cases {
		require.Equal(t, c.want, slugify(c.in), c.in)
	}
}

func TestSanitizeHTMLTags(t *testing.T) {
	cases := []struct{ in, want string }{
		{"plain text", "plain text"},
		{"<b>bold</b>", "&lt;b&gt;bold&lt;/b&gt;"},
		{"use {braces} here", "use &lbrace;braces&rbrace; here"},
		{"keep `a<b>c` inline", "keep `a<b>c` inline"},
	}
	for _, c := range cases {
		require.Equal(t, c.want, sanitizeHTMLTags(c.in), c.in)
	}

	fenced := "before\n```\n<div>{raw}\n```\nafter {x}"
	out := sanitizeHTMLTags(fenced)
	require.Contains(t, out, "<div>{raw}")
	require.Contains(t, out, "after &lbrace;x&rbrace;")
}

func TestAdjustHeadingLevels(t *testing.T) {
	require.Equal(t, "## Top", adjustHeadingLevels("# Top"))
	require.Equal(t, "###### Deep", adjustHeadingLevels("###### Deep"))
	require.Equal(t, "no heading", adjustHeadingLevels("no heading"))
	require.Equal(t, "intro\n## Nested\ntail", adjustHeadingLevels("intro\n# Nested\ntail"))
}

func TestIntegrationFilename(t *testing.T) {
	require.Equal(t, "TestTool", integrationFilename(fakeIntegration{
		label: "Test Tool",
		name:  "test_tool",
	}))
	require.Equal(t, "fallback-name", integrationFilename(fakeIntegration{
		label: "  ",
		name:  "Fallback Name",
	}))
}

func TestRenderIntegrationDoc(t *testing.T) {
	integration := fakeIntegration{
		name:         "test_tool",
		label:        "Test Tool",
		description:  "Connects to <b>things</b>",
		instructions: "## Setup\nRun it",
		triggers: []core.Trigger{
			fakeTrigger{
				name:          "on_push",
				label:         "On Push",
				description:   "Fires on push events",
				documentation: "# Details\nBody text",
				exampleData:   map[string]any{"event": "push"},
			},
		},
		actions: []core.Action{
			fakeAction{
				name:          "get_stuff",
				label:         "Get \"Stuff\"",
				description:   "Fetches stuff",
				exampleOutput: map[string]any{"result": 42},
			},
		},
	}

	doc, err := renderIntegrationDoc(integration)
	require.NoError(t, err)
	out := string(doc)

	require.Contains(t, out, "---\ntitle: \"Test Tool\"\n---")
	require.Contains(t, out, "import { CardGrid, LinkCard }")
	require.Contains(t, out, "Connects to &lt;b&gt;things&lt;/b&gt;")
	require.Contains(t, out, `<LinkCard title="On Push" href="#on-push"`)
	require.Contains(t, out, `title="Get \"Stuff\""`)
	require.Contains(t, out, `href="#get-"stuff""`)
	require.Contains(t, out, "**Trigger key:** `on_push`")
	require.Contains(t, out, "**Component key:** `get_stuff`")
	require.Contains(t, out, "\n## Details\n")
	require.Contains(t, out, "### Example Output")
	require.Contains(t, out, "\"result\": 42")
	require.Contains(t, out, "### Example Data")
	require.Contains(t, out, "## Instructions")
	require.Contains(t, out, "## Setup")
	require.NotContains(t, out, "sidebar:")
}

func TestRenderCoreComponentsDocEmpty(t *testing.T) {
	doc, err := renderCoreComponentsDoc(nil, nil)
	require.NoError(t, err)
	require.Nil(t, doc)
}

func TestRenderCoreComponentsDocSetsSidebarOrder(t *testing.T) {
	doc, err := renderCoreComponentsDoc(
		[]core.Action{fakeAction{name: "http", label: "HTTP"}},
		nil,
	)
	require.NoError(t, err)
	out := string(doc)
	require.Contains(t, out, "sidebar:\n  order: 1")
	require.Contains(t, out, "title: \"Core\"")
	require.NotContains(t, out, "## Triggers")
	require.Contains(t, out, "## Actions")
}

func TestRenderComponentIndex(t *testing.T) {
	actions := []core.Action{
		fakeAction{name: "get_stuff", label: "Get Stuff", description: "Rows | split"},
	}
	triggers := []core.Trigger{
		fakeTrigger{name: "on_push", label: "On Push", description: "Fires\noften"},
	}
	integrations := []core.Integration{
		fakeIntegration{label: "Test Tool", actions: actions, triggers: triggers},
	}

	out := string(renderComponentIndex(actions, triggers, integrations))

	require.Contains(t, out, "# SuperPlane Component Index")
	require.Contains(t, out, "| Get Stuff | `get_stuff` | Rows \\| split |")
	require.Contains(t, out, "| On Push | `on_push` | Fires often |")
	require.Contains(t, out, "## Test Tool Triggers")
	require.Contains(t, out, "## Test Tool Actions")
}

func TestWriteFiles(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, WriteFiles(dir))

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.NotEmpty(t, entries)

	for _, entry := range entries {
		require.True(t, strings.HasSuffix(entry.Name(), ".mdx"), entry.Name())
		content, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		require.NoError(t, err)
		require.NotEmpty(t, content)
	}
}
