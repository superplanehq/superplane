package notion

import (
	"fmt"
	"math"
	"net/http"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type Search struct{}

type SearchSpec struct {
	Query         string   `json:"query" mapstructure:"query"`
	ObjectType    string   `json:"objectType" mapstructure:"objectType"`
	PageSize      *float64 `json:"pageSize,omitempty" mapstructure:"pageSize"`
	StartCursor   string   `json:"startCursor" mapstructure:"startCursor"`
	SortDirection string   `json:"sortDirection" mapstructure:"sortDirection"`
}

func (s *Search) Name() string        { return "notion.search" }
func (s *Search) Label() string       { return "Search" }
func (s *Search) Description() string { return "Search shared Notion pages and data sources by title" }
func (s *Search) Icon() string        { return "notion" }
func (s *Search) Color() string       { return "gray" }
func (s *Search) Documentation() string {
	return `Search shared page and data source titles. Search does not match page content or query data source entries.

## Configuration

- **Query**: Enter title text, or leave empty to list shared objects.
- **Object Type**: Select pages, data sources, or both.
- **Page Size**: Return 1–100 results. The default is 100.
- **Start Cursor**: Use next_cursor from the previous output. Keep the query and filters unchanged.
- **Sort Direction**: Optionally sort by the last edit time.

## Output

Each execution returns one page of results in data, with results, has_more, and next_cursor. Empty results are successful.

The connection requires **Read content**. Newly shared or created content may take time to appear. Requests do not retry automatically.`
}

func (s *Search) Configuration() []configuration.Field {
	minimum, maximum := 1, 100
	return []configuration.Field{
		{Name: "query", Label: "Query", Type: configuration.FieldTypeString, Description: "Text to find in titles. Leave empty to list shared objects."},
		{Name: "objectType", Label: "Object Type", Type: configuration.FieldTypeSelect, Default: "all", TypeOptions: &configuration.TypeOptions{Select: &configuration.SelectTypeOptions{Options: []configuration.FieldOption{
			{Label: "All", Value: "all"}, {Label: "Page", Value: "page"}, {Label: "Data Source", Value: "data_source"},
		}}}},
		{Name: "pageSize", Label: "Page Size", Type: configuration.FieldTypeNumber, Default: 100, TypeOptions: &configuration.TypeOptions{Number: &configuration.NumberTypeOptions{Min: &minimum, Max: &maximum}}},
		{Name: "startCursor", Label: "Start Cursor", Type: configuration.FieldTypeString, Description: "The next_cursor from the previous search. Leave empty for the first page."},
		{Name: "sortDirection", Label: "Sort Direction", Type: configuration.FieldTypeSelect, Description: "Sort results by the last edit time", TypeOptions: &configuration.TypeOptions{Select: &configuration.SelectTypeOptions{Options: []configuration.FieldOption{
			{Label: "Ascending", Value: "ascending"}, {Label: "Descending", Value: "descending"},
		}}}},
	}
}

func (s *Search) Setup(ctx core.SetupContext) error {
	_, err := searchConfiguration(ctx.Configuration, true)
	return err
}

func searchConfiguration(configuration any, setup bool) (SearchSpec, error) {
	if setup {
		// Numeric expressions cannot decode into a number until execution.
		if values, ok := configuration.(map[string]any); ok {
			if value, ok := values["pageSize"].(string); ok && strings.Contains(value, "{{") {
				copy := make(map[string]any, len(values))
				for key, value := range values {
					copy[key] = value
				}
				delete(copy, "pageSize")
				configuration = copy
			}
		}
	}
	var spec SearchSpec
	if err := mapstructure.Decode(configuration, &spec); err != nil {
		return spec, fmt.Errorf("invalid search configuration: %w", err)
	}
	if spec.ObjectType == "" {
		spec.ObjectType = "all"
	}
	if spec.ObjectType != "all" && spec.ObjectType != "page" && spec.ObjectType != "data_source" && !(setup && strings.Contains(spec.ObjectType, "{{")) {
		return spec, fmt.Errorf("object type must be all, page, or data_source")
	}
	if spec.SortDirection != "" && spec.SortDirection != "ascending" && spec.SortDirection != "descending" && !(setup && strings.Contains(spec.SortDirection, "{{")) {
		return spec, fmt.Errorf("sort direction must be ascending or descending")
	}
	if spec.PageSize != nil && (*spec.PageSize < 1 || *spec.PageSize > 100 || math.Trunc(*spec.PageSize) != *spec.PageSize) {
		return spec, fmt.Errorf("page size must be an integer from 1 to 100")
	}
	return spec, nil
}

func (s *Search) Execute(ctx core.ExecutionContext) error {
	spec, err := searchConfiguration(ctx.Configuration, false)
	if err != nil {
		return err
	}
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}
	request := searchRequest{Query: spec.Query, StartCursor: spec.StartCursor, PageSize: 100}
	if spec.PageSize != nil {
		request.PageSize = int(*spec.PageSize)
	}
	if spec.ObjectType != "all" {
		request.Filter = &searchFilter{Property: "object", Value: spec.ObjectType}
	}
	if spec.SortDirection != "" {
		request.Sort = &searchSort{Timestamp: "last_edited_time", Direction: spec.SortDirection}
	}
	result, err := client.Search(request)
	if err != nil {
		return err
	}
	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, "notion.search", []any{result})
}

func (s *Search) OutputChannels(any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}
func (s *Search) Cancel(core.ExecutionContext) error      { return nil }
func (s *Search) Cleanup(core.SetupContext) error         { return nil }
func (s *Search) Hooks() []core.Hook                      { return nil }
func (s *Search) HandleHook(core.ActionHookContext) error { return nil }
func (s *Search) HandleWebhook(core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}
