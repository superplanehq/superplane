package render

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const ListLogsPayloadType = "render.logs"

type ListLogs struct{}

type ListLogsConfiguration struct {
	Resources []string `json:"resources" mapstructure:"resources"`
	Levels    []string `json:"levels" mapstructure:"levels"`
	Types     []string `json:"types" mapstructure:"types"`
	Text      []string `json:"text" mapstructure:"text"`
	Paths     []string `json:"paths" mapstructure:"paths"`
	StartTime string   `json:"startTime" mapstructure:"startTime"`
	EndTime   string   `json:"endTime" mapstructure:"endTime"`
	Direction string   `json:"direction" mapstructure:"direction"`
	Limit     int      `json:"limit" mapstructure:"limit"`
}

func (c *ListLogs) Name() string { return "render.listLogs" }

func (c *ListLogs) Label() string { return "List Logs" }

func (c *ListLogs) Description() string {
	return "Query recent Render logs for one or more resources"
}

func (c *ListLogs) Documentation() string {
	return `Query recent Render logs for services, jobs, Postgres, and Key Value resources.`
}

func (c *ListLogs) Icon() string { return "scroll-text" }

func (c *ListLogs) Color() string { return "gray" }

func (c *ListLogs) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *ListLogs) Configuration() []configuration.Field {
	return []configuration.Field{
		stringListField("resources", "Resources", true, "Render resource IDs to query"),
		stringListField("levels", "Levels", false, "Optional log levels"),
		stringListField("types", "Types", false, "Optional log types, for example app or request"),
		stringListField("text", "Text", false, "Optional text filters"),
		stringListField("paths", "Paths", false, "Optional request path filters"),
		{Name: "startTime", Label: "Start Time", Type: configuration.FieldTypeString, Required: false},
		{Name: "endTime", Label: "End Time", Type: configuration.FieldTypeString, Required: false},
		{Name: "direction", Label: "Direction", Type: configuration.FieldTypeString, Required: false, Default: "backward"},
		{Name: "limit", Label: "Limit", Type: configuration.FieldTypeNumber, Required: false, Default: 50},
	}
}

func decodeListLogsConfiguration(configuration any) (ListLogsConfiguration, error) {
	spec := ListLogsConfiguration{Direction: "backward", Limit: 50}
	if err := decodeActionConfiguration(configuration, &spec); err != nil {
		return ListLogsConfiguration{}, err
	}
	spec.Resources = cleanStringList(spec.Resources)
	spec.Levels = cleanStringList(spec.Levels)
	spec.Types = cleanStringList(spec.Types)
	spec.Text = cleanStringList(spec.Text)
	spec.Paths = cleanStringList(spec.Paths)
	spec.StartTime = strings.TrimSpace(spec.StartTime)
	spec.EndTime = strings.TrimSpace(spec.EndTime)
	spec.Direction = strings.TrimSpace(spec.Direction)
	if len(spec.Resources) == 0 {
		return ListLogsConfiguration{}, fmt.Errorf("at least one resource is required")
	}
	if spec.Limit < 1 {
		spec.Limit = 50
	}
	return spec, nil
}

func (c *ListLogs) Setup(ctx core.SetupContext) error {
	_, err := decodeListLogsConfiguration(ctx.Configuration)
	return err
}

func (c *ListLogs) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeListLogsConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	workspaceID, err := workspaceIDForIntegration(client, ctx.Integration)
	if err != nil {
		return err
	}

	response, err := client.ListLogs(workspaceID, LogQuery{
		Resources: spec.Resources,
		Levels:    spec.Levels,
		Types:     spec.Types,
		Text:      spec.Text,
		Paths:     spec.Paths,
		StartTime: spec.StartTime,
		EndTime:   spec.EndTime,
		Direction: spec.Direction,
		Limit:     spec.Limit,
	})
	if err != nil {
		return err
	}

	errorCount := 0
	for _, item := range response.Logs {
		level := strings.ToLower(readString(item["level"]))
		status := readString(item["statusCode"])
		if level == "error" || strings.HasPrefix(status, "5") {
			errorCount++
		}
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, ListLogsPayloadType, []any{map[string]any{
		"resources":     spec.Resources,
		"count":         len(response.Logs),
		"errorCount":    errorCount,
		"hasMore":       response.HasMore,
		"nextStartTime": response.NextStartTime,
		"nextEndTime":   response.NextEndTime,
		"logs":          response.Logs,
	}})
}

func (c *ListLogs) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}
func (c *ListLogs) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}
func (c *ListLogs) Cancel(ctx core.ExecutionContext) error      { return nil }
func (c *ListLogs) Cleanup(ctx core.SetupContext) error         { return nil }
func (c *ListLogs) Hooks() []core.Hook                          { return []core.Hook{} }
func (c *ListLogs) HandleHook(ctx core.ActionHookContext) error { return nil }
