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

//go:embed example_output_update_synthetic_check.json
var exampleOutputUpdateSyntheticCheckBytes []byte

var exampleOutputUpdateSyntheticCheckOnce sync.Once
var exampleOutputUpdateSyntheticCheck map[string]any

func (c *QueryPrometheus) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputQueryPrometheusOnce, exampleOutputQueryPrometheusBytes, &exampleOutputQueryPrometheus)
}

func (c *ListIssues) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputListIssuesOnce, exampleOutputListIssuesBytes, &exampleOutputListIssues)
}

// ExampleOutput returns sample output data for Update Synthetic Check.
func (c *UpdateSyntheticCheck) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputUpdateSyntheticCheckOnce, exampleOutputUpdateSyntheticCheckBytes, &exampleOutputUpdateSyntheticCheck)
}
