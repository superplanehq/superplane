package dash0

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type QueryPrometheus struct{}

type QueryPrometheusSpec struct {
	Query   string  `json:"query"`
	Dataset string  `json:"dataset"`
	Type    string  `json:"type"` // "instant" or "range"
	Start   *string `json:"start,omitempty"` // For range queries, e.g., "now-5m"
	End     *string `json:"end,omitempty"`   // For range queries, e.g., "now"
	Step    *string `json:"step,omitempty"` // For range queries, e.g., "15s"
}

func (q *QueryPrometheus) Name() string {
	return "dash0.queryPrometheus"
}

func (q *QueryPrometheus) Label() string {
	return "Query Prometheus"
}

func (q *QueryPrometheus) Description() string {
	return "Execute a PromQL query against Dash0 Prometheus API and return the response data"
}

func (q *QueryPrometheus) Icon() string {
	return "database"
}

func (q *QueryPrometheus) Color() string {
	return "blue"
}

func (q *QueryPrometheus) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (q *QueryPrometheus) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "query",
			Label:       "PromQL Query",
			Type:        configuration.FieldTypeText,
			Required:    true,
			Description: "The PromQL (Prometheus Query Language) query to execute",
			Placeholder: "sum by (service_name) (increase({otel_metric_name = \"dash0.spans\"}[15s])) > 0",
		},
		{
			Name:        "dataset",
			Label:       "Dataset",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Default:     "default",
			Description: "The dataset to query",
		},
		{
			Name:     "type",
			Label:    "Query Type",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  "instant",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Instant", Value: "instant"},
						{Label: "Range", Value: "range"},
					},
				},
			},
		},
		{
			Name:        "start",
			Label:       "Start Time",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Start time for range queries (e.g., 'now-5m', '2024-01-01T00:00:00Z')",
			Placeholder: "now-5m",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "type", Values: []string{"range"}},
			},
		},
		{
			Name:        "end",
			Label:       "End Time",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "End time for range queries (e.g., 'now', '2024-01-01T01:00:00Z')",
			Placeholder: "now",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "type", Values: []string{"range"}},
			},
		},
		{
			Name:        "step",
			Label:       "Step",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Query resolution step width for range queries (e.g., '15s', '1m', '5m')",
			Placeholder: "15s",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "type", Values: []string{"range"}},
			},
		},
	}
}

func (q *QueryPrometheus) Setup(ctx core.SetupContext) error {
	spec := QueryPrometheusSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.Query == "" {
		return errors.New("query is required")
	}

	if len(strings.TrimSpace(spec.Query)) == 0 {
		return errors.New("query cannot be empty")
	}

	if spec.Dataset == "" {
		return errors.New("dataset is required")
	}

	if spec.Type == "range" {
		if spec.Start == nil || *spec.Start == "" {
			return errors.New("start is required for range queries")
		}
		if spec.End == nil || *spec.End == "" {
			return errors.New("end is required for range queries")
		}
		if spec.Step == nil || *spec.Step == "" {
			return errors.New("step is required for range queries")
		}
	}

	return nil
}

func (q *QueryPrometheus) Execute(ctx core.ExecutionContext) error {
	spec := QueryPrometheusSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.AppInstallation)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	var data map[string]any
	if spec.Type == "range" {
		data, err = client.ExecutePrometheusRangeQuery(spec.Query, spec.Dataset, *spec.Start, *spec.End, *spec.Step)
	} else {
		data, err = client.ExecutePrometheusInstantQuery(spec.Query, spec.Dataset)
	}

	if err != nil {
		return fmt.Errorf("failed to execute Prometheus query: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"dash0.prometheus.response",
		[]any{data},
	)
}

func (q *QueryPrometheus) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (q *QueryPrometheus) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (q *QueryPrometheus) Actions() []core.Action {
	return []core.Action{}
}

func (q *QueryPrometheus) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (q *QueryPrometheus) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}
