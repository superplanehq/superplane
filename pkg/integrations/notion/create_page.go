package notion

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type CreatePage struct{}

type CreatePageSpec struct {
	ParentType string `json:"parentType" mapstructure:"parentType"`
	ParentID   string `json:"parentId" mapstructure:"parentId"`
	Title      string `json:"title" mapstructure:"title"`
	Properties *any   `json:"properties,omitempty" mapstructure:"properties"`
	Children   any    `json:"children,omitempty" mapstructure:"children"`
}

func (c *CreatePage) Name() string        { return "notion.createPage" }
func (c *CreatePage) Label() string       { return "Create Page" }
func (c *CreatePage) Description() string { return "Create a page under a Notion page or data source" }
func (c *CreatePage) Icon() string        { return "notion" }
func (c *CreatePage) Color() string       { return "gray" }
func (c *CreatePage) Documentation() string {
	return `Create Page creates one Notion page under an existing page or data source.

## Configuration

- **Parent type**: Select Page for a child page or Data Source for a database entry.
- **Parent ID**: Enter the page ID or data source ID. Expressions can supply the ID at execution time.
- **Title**: Enter the page title. The maximum length is 2,000 characters.
- **Properties**: For a data source, optionally enter a JSON object with additional page properties.
- **Children**: Optionally enter a JSON array of Notion blocks for the page content.

Use a data source ID, not a database ID. In Notion, open **Manage data sources** and copy the data source ID.
The component reads the current data source schema to find its title property. It supports custom title property names.
Set the title with the Title field. Do not include the title property in Properties.

Properties must match the selected data source schema. For example: {"Status":{"select":{"name":"Open"}}}.
Properties and Children also accept expressions that return an object or array, such as {{ previous().data.children }}.
Children must use the [Notion block format](https://developers.notion.com/reference/block).
For example: [{"object":"block","type":"paragraph","paragraph":{"rich_text":[{"type":"text","text":{"content":"Runbook steps"}}]}}].
Requests can contain at most 100 top-level blocks and 500 KB of JSON. Notion validates other content and schema limits.

## Permissions and output

Share the parent with the connection. Enable Insert content. Data source schema reads also need Read content.
The output contains the native page in data, including id, url, parent, and properties.

The component does not retry creation automatically. After a network or server error, the page may already exist.
Check Notion before you try again. A repeated run can create another page.`
}

func (c *CreatePage) Configuration() []configuration.Field {
	return []configuration.Field{
		{Name: "parentType", Label: "Parent Type", Type: configuration.FieldTypeSelect, Required: true, Default: "page", TypeOptions: &configuration.TypeOptions{Select: &configuration.SelectTypeOptions{Options: []configuration.FieldOption{
			{Label: "Page", Value: "page"}, {Label: "Data Source", Value: "data_source"},
		}}}},
		{Name: "parentId", Label: "Parent ID", Type: configuration.FieldTypeString, Required: true, Description: "The page ID or data source ID. Do not use a database ID."},
		{Name: "title", Label: "Title", Type: configuration.FieldTypeString, Required: true, Description: "The title of the new page"},
		{Name: "properties", Label: "Properties", Type: configuration.FieldTypeObject, Description: "A JSON object of data source properties. Set the title in the Title field.", VisibilityConditions: []configuration.VisibilityCondition{{Field: "parentType", Values: []string{"data_source"}}}},
		{Name: "children", Label: "Children", Type: configuration.FieldTypeObject, Description: "A JSON array of Notion blocks for the page content"},
	}
}

func (c *CreatePage) Setup(ctx core.SetupContext) error {
	_, _, _, err := createPageConfiguration(ctx.Configuration, true)
	return err
}

func createPageConfiguration(configuration any, setup bool) (CreatePageSpec, map[string]any, []map[string]any, error) {
	var spec CreatePageSpec
	if err := mapstructure.Decode(configuration, &spec); err != nil {
		return spec, nil, nil, fmt.Errorf("invalid page configuration: %w", err)
	}
	if spec.ParentType == "" {
		spec.ParentType = "page"
	}
	if spec.ParentType != "page" && spec.ParentType != "data_source" && !(setup && strings.Contains(spec.ParentType, "{{")) {
		return spec, nil, nil, fmt.Errorf("parent type must be page or data_source")
	}
	if !(setup && strings.Contains(spec.ParentID, "{{")) {
		id, err := uuid.Parse(strings.TrimSpace(spec.ParentID))
		if err != nil || id == uuid.Nil {
			return spec, nil, nil, fmt.Errorf("parent ID must be a page or data source UUID")
		}
		spec.ParentID = id.String()
	}
	if strings.TrimSpace(spec.Title) == "" {
		return spec, nil, nil, fmt.Errorf("title is required")
	}
	if !(setup && strings.Contains(spec.Title, "{{")) && utf8.RuneCountInString(spec.Title) > 2000 {
		return spec, nil, nil, fmt.Errorf("title must contain at most 2,000 characters")
	}
	properties := map[string]any{}
	if spec.ParentType != "page" && spec.Properties != nil {
		data, err := pageJSON(*spec.Properties, setup)
		if err != nil {
			return spec, nil, nil, fmt.Errorf("invalid properties: %w", err)
		}
		if len(data) > 0 {
			if err := json.Unmarshal(data, &properties); err != nil || properties == nil {
				return spec, nil, nil, fmt.Errorf("properties must be a JSON object")
			}
			for _, value := range properties {
				if _, ok := value.(map[string]any); !ok {
					return spec, nil, nil, fmt.Errorf("each property must be a JSON object in the Notion property format")
				}
			}
		}
	}
	var children []map[string]any
	data, err := pageJSON(spec.Children, setup)
	if err != nil {
		return spec, nil, nil, fmt.Errorf("invalid children: %w", err)
	}
	if len(data) > 0 {
		if err := json.Unmarshal(data, &children); err != nil || children == nil {
			return spec, nil, nil, fmt.Errorf("children must be a JSON array of blocks")
		}
		if len(children) > 100 {
			return spec, nil, nil, fmt.Errorf("children must contain at most 100 top-level blocks")
		}
		for _, block := range children {
			if len(block) == 0 {
				return spec, nil, nil, fmt.Errorf("each child must be a Notion block object")
			}
		}
	}
	return spec, properties, children, nil
}

// JSON fields arrive as either editor text or native values after expression resolution.
func pageJSON(value any, setup bool) ([]byte, error) {
	if value == nil {
		return nil, nil
	}
	var data []byte
	if text, ok := value.(string); ok {
		if strings.TrimSpace(text) == "" {
			return nil, nil
		}
		data = []byte(text)
	} else {
		var err error
		data, err = json.Marshal(value)
		if err != nil {
			return nil, fmt.Errorf("value must be valid JSON")
		}
	}
	if setup && strings.Contains(string(data), "{{") {
		return nil, nil
	}
	if len(data) > maxRequestBytes {
		return nil, fmt.Errorf("value exceeds the 500 KB request limit")
	}
	return data, nil
}

func (c *CreatePage) Execute(ctx core.ExecutionContext) error {
	spec, properties, children, err := createPageConfiguration(ctx.Configuration, false)
	if err != nil {
		return err
	}
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}
	titleKey := "title"
	if spec.ParentType == "data_source" {
		source, err := client.DataSource(spec.ParentID)
		if err != nil {
			return fmt.Errorf("could not read the data source schema: %w", err)
		}
		name, id, err := titleProperty(source)
		if err != nil {
			return err
		}
		if _, ok := properties[name]; ok {
			return fmt.Errorf("set the title in the Title field, not Properties")
		}
		if _, ok := properties[id]; ok {
			return fmt.Errorf("set the title in the Title field, not Properties")
		}
		titleKey = id
	}
	properties[titleKey] = map[string]any{"title": []any{map[string]any{"type": "text", "text": map[string]any{"content": spec.Title}}}}
	page, err := client.CreatePage(createPageRequest{
		Parent: map[string]string{spec.ParentType + "_id": spec.ParentID}, Properties: properties, Children: children,
	})
	if err != nil {
		return err
	}
	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, "notion.page", []any{page})
}

func titleProperty(source map[string]any) (string, string, error) {
	properties, ok := source["properties"].(map[string]any)
	if !ok || source["object"] != "data_source" {
		return "", "", fmt.Errorf("Notion did not return a data source schema. Check the data source ID")
	}
	for name, value := range properties {
		property, ok := value.(map[string]any)
		if !ok || property["type"] != "title" {
			continue
		}
		id, _ := property["id"].(string)
		if id == "" {
			id = name
		}
		return name, id, nil
	}
	return "", "", fmt.Errorf("the data source has no title property")
}

func (c *CreatePage) OutputChannels(any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}
func (c *CreatePage) Cancel(core.ExecutionContext) error      { return nil }
func (c *CreatePage) Cleanup(core.SetupContext) error         { return nil }
func (c *CreatePage) Hooks() []core.Hook                      { return nil }
func (c *CreatePage) HandleHook(core.ActionHookContext) error { return nil }
func (c *CreatePage) HandleWebhook(core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}
