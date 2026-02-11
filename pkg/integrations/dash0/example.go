package dash0

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_query_prometheus.json
var exampleOutputQueryPrometheusBytes []byte

var exampleOutputQueryPrometheusOnce sync.Once
var exampleOutputQueryPrometheus map[string]any

//go:embed example_output_list_issues.json
var exampleOutputListIssuesBytes []byte

var exampleOutputListIssuesOnce sync.Once
var exampleOutputListIssues map[string]any

//go:embed example_output_send_log_event.json
var exampleOutputSendLogEventBytes []byte

var exampleOutputSendLogEventOnce sync.Once
var exampleOutputSendLogEvent map[string]any

//go:embed example_output_get_check_details.json
var exampleOutputGetCheckDetailsBytes []byte

var exampleOutputGetCheckDetailsOnce sync.Once
var exampleOutputGetCheckDetails map[string]any

//go:embed example_output_create_synthetic_check.json
var exampleOutputCreateSyntheticCheckBytes []byte

var exampleOutputCreateSyntheticCheckOnce sync.Once
var exampleOutputCreateSyntheticCheck map[string]any

func (c *QueryPrometheus) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputQueryPrometheusOnce, exampleOutputQueryPrometheusBytes, &exampleOutputQueryPrometheus)
}

func (c *ListIssues) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputListIssuesOnce, exampleOutputListIssuesBytes, &exampleOutputListIssues)
}

// ExampleOutput returns sample output data for Send Log Event.
func (c *SendLogEvent) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputSendLogEventOnce, exampleOutputSendLogEventBytes, &exampleOutputSendLogEvent)
}

// ExampleOutput returns sample output data for Get Check Details.
func (c *GetCheckDetails) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetCheckDetailsOnce, exampleOutputGetCheckDetailsBytes, &exampleOutputGetCheckDetails)
}

// ExampleOutput returns sample output data for Create Synthetic Check.
func (c *CreateSyntheticCheck) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateSyntheticCheckOnce, exampleOutputCreateSyntheticCheckBytes, &exampleOutputCreateSyntheticCheck)
}
