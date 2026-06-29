package memory

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/superplanehq/superplane/pkg/cli/commands/apps/common"
	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type listCommand struct {
	namespace *string
}

func (c *listCommand) Execute(ctx core.CommandContext) error {
	if len(ctx.Args) > 1 {
		return fmt.Errorf("list accepts at most one positional argument")
	}

	appArg := ""
	if len(ctx.Args) == 1 {
		appArg = strings.TrimSpace(ctx.Args[0])
	}

	appID, err := common.ResolveAppNameOrIDArg(ctx, appArg)
	if err != nil {
		return err
	}

	response, _, err := ctx.API.CanvasAPI.CanvasesListCanvasMemories(ctx.Context, appID).Execute()
	if err != nil {
		return err
	}

	memories := filterByNamespace(response.GetItems(), c.namespaceValue())
	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(memories)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		return renderMemoryListText(stdout, memories)
	})
}

func (c *listCommand) namespaceValue() string {
	if c.namespace == nil {
		return ""
	}

	return strings.TrimSpace(*c.namespace)
}

func filterByNamespace(memories []openapi_client.CanvasesCanvasMemory, namespace string) []openapi_client.CanvasesCanvasMemory {
	if namespace == "" {
		return memories
	}

	filtered := make([]openapi_client.CanvasesCanvasMemory, 0, len(memories))
	for _, memory := range memories {
		if memory.GetNamespace() == namespace {
			filtered = append(filtered, memory)
		}
	}

	return filtered
}

func renderMemoryListText(stdout io.Writer, memories []openapi_client.CanvasesCanvasMemory) error {
	if len(memories) == 0 {
		_, err := fmt.Fprintln(stdout, "No memory records found.")
		return err
	}

	groups := groupByNamespace(memories)
	for i, group := range groups {
		if len(groups) > 1 {
			if i > 0 {
				_, _ = fmt.Fprintln(stdout)
			}
			_, _ = fmt.Fprintf(stdout, "Namespace: %s\n", group.namespace)
		}

		if err := renderNamespaceTable(stdout, group.memories); err != nil {
			return err
		}
	}

	return nil
}

type namespaceGroup struct {
	namespace string
	memories  []openapi_client.CanvasesCanvasMemory
}

func groupByNamespace(memories []openapi_client.CanvasesCanvasMemory) []namespaceGroup {
	groupsByNamespace := map[string]int{}
	groups := make([]namespaceGroup, 0)

	for _, memory := range memories {
		namespace := memory.GetNamespace()
		if namespace == "" {
			namespace = "(no namespace)"
		}

		index, ok := groupsByNamespace[namespace]
		if !ok {
			groupsByNamespace[namespace] = len(groups)
			groups = append(groups, namespaceGroup{namespace: namespace})
			index = len(groups) - 1
		}

		groups[index].memories = append(groups[index].memories, memory)
	}

	return groups
}

func renderNamespaceTable(stdout io.Writer, memories []openapi_client.CanvasesCanvasMemory) error {
	columns := collectValueColumns(memories)
	if len(columns) == 0 {
		columns = []string{"VALUE"}
	}

	writer := tabwriter.NewWriter(stdout, 0, 8, 2, ' ', 0)
	_, _ = fmt.Fprintln(writer, strings.Join(columns, "\t"))

	for _, memory := range memories {
		values := memory.GetValues()
		row := make([]string, 0, len(columns))
		for _, column := range columns {
			value, ok := values[column]
			if !ok {
				row = append(row, "")
				continue
			}

			formatted, err := formatValue(value)
			if err != nil {
				return err
			}

			row = append(row, formatted)
		}

		_, _ = fmt.Fprintln(writer, strings.Join(row, "\t"))
	}

	return writer.Flush()
}

func collectValueColumns(memories []openapi_client.CanvasesCanvasMemory) []string {
	columnsByName := map[string]bool{}
	for _, memory := range memories {
		for column := range memory.GetValues() {
			columnsByName[column] = true
		}
	}

	columns := make([]string, 0, len(columnsByName))
	for column := range columnsByName {
		columns = append(columns, column)
	}
	sort.Strings(columns)

	return columns
}

func formatValue(value any) (string, error) {
	switch typed := value.(type) {
	case nil:
		return "", nil
	case string:
		return typed, nil
	default:
		payload, err := json.Marshal(typed)
		if err != nil {
			return "", fmt.Errorf("failed to format memory value: %w", err)
		}

		return string(payload), nil
	}
}
