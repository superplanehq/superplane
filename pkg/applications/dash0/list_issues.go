package dash0

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type ListIssues struct{}

type ListIssuesSpec struct {
	Dataset string `json:"dataset"`
}

func (l *ListIssues) Name() string {
	return "dash0.listIssues"
}

func (l *ListIssues) Label() string {
	return "List Issues"
}

func (l *ListIssues) Description() string {
	return "Query Dash0 to get a list of all current issues using the metric dash0.issue.status"
}

func (l *ListIssues) Icon() string {
	return "alert-triangle"
}

func (l *ListIssues) Color() string {
	return "orange"
}

func (l *ListIssues) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (l *ListIssues) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "dataset",
			Label:       "Dataset",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Default:     "default",
			Description: "The dataset to query",
		},
	}
}

func (l *ListIssues) Setup(ctx core.SetupContext) error {
	spec := ListIssuesSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.Dataset == "" {
		return fmt.Errorf("dataset is required")
	}

	return nil
}

func (l *ListIssues) Execute(ctx core.ExecutionContext) error {
	spec := ListIssuesSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.AppInstallation)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	// Execute the query to get all current issues
	query := `{otel_metric_name="dash0.issue.status"} >= 1`
	data, err := client.ExecutePrometheusInstantQuery(query, spec.Dataset)
	if err != nil {
		return fmt.Errorf("failed to execute Prometheus query: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"dash0.issues.list",
		[]any{data},
	)
}

func (l *ListIssues) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (l *ListIssues) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (l *ListIssues) Actions() []core.Action {
	return []core.Action{}
}

func (l *ListIssues) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (l *ListIssues) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}
