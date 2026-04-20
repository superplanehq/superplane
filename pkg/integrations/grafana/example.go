package grafana

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_query_data_source.json
var exampleOutputQueryDataSourceBytes []byte

//go:embed example_output_create_alert_rule.json
var exampleOutputCreateAlertRuleBytes []byte

//go:embed example_output_get_alert_rule.json
var exampleOutputGetAlertRuleBytes []byte

//go:embed example_output_update_alert_rule.json
var exampleOutputUpdateAlertRuleBytes []byte

//go:embed example_output_delete_alert_rule.json
var exampleOutputDeleteAlertRuleBytes []byte

//go:embed example_output_list_alert_rules.json
var exampleOutputListAlertRulesBytes []byte

//go:embed example_data_on_alert_firing.json
var exampleDataOnAlertFiringBytes []byte

//go:embed example_output_get_dashboard.json
var exampleOutputGetDashboardBytes []byte

//go:embed example_output_render_panel.json
var exampleOutputRenderPanelBytes []byte

//go:embed example_output_list_silences.json
var exampleOutputListSilencesBytes []byte

//go:embed example_output_get_silence.json
var exampleOutputGetSilenceBytes []byte

//go:embed example_output_create_silence.json
var exampleOutputCreateSilenceBytes []byte

//go:embed example_output_delete_silence.json
var exampleOutputDeleteSilenceBytes []byte

//go:embed example_output_create_annotation.json
var exampleOutputCreateAnnotationBytes []byte

//go:embed example_output_list_annotations.json
var exampleOutputListAnnotationsBytes []byte

//go:embed example_output_delete_annotation.json
var exampleOutputDeleteAnnotationBytes []byte

//go:embed example_output_create_http_synthetic_check.json
var exampleOutputCreateHTTPSyntheticCheckBytes []byte

//go:embed example_output_get_http_synthetic_check.json
var exampleOutputGetHTTPSyntheticCheckBytes []byte

//go:embed example_output_update_http_synthetic_check.json
var exampleOutputUpdateHTTPSyntheticCheckBytes []byte

//go:embed example_output_delete_http_synthetic_check.json
var exampleOutputDeleteHTTPSyntheticCheckBytes []byte

var exampleOutputQueryDataSourceOnce sync.Once
var exampleOutputQueryDataSource map[string]any

var exampleOutputCreateAlertRuleOnce sync.Once
var exampleOutputCreateAlertRule map[string]any

var exampleOutputGetAlertRuleOnce sync.Once
var exampleOutputGetAlertRule map[string]any

var exampleOutputUpdateAlertRuleOnce sync.Once
var exampleOutputUpdateAlertRule map[string]any

var exampleOutputDeleteAlertRuleOnce sync.Once
var exampleOutputDeleteAlertRule map[string]any

var exampleOutputListAlertRulesOnce sync.Once
var exampleOutputListAlertRules map[string]any

var exampleDataOnAlertFiringOnce sync.Once
var exampleDataOnAlertFiring map[string]any

var exampleOutputGetDashboardOnce sync.Once
var exampleOutputGetDashboard map[string]any

var exampleOutputRenderPanelOnce sync.Once
var exampleOutputRenderPanel map[string]any

var exampleOutputListSilencesOnce sync.Once
var exampleOutputListSilences map[string]any

var exampleOutputGetSilenceOnce sync.Once
var exampleOutputGetSilence map[string]any

var exampleOutputCreateSilenceOnce sync.Once
var exampleOutputCreateSilence map[string]any

var exampleOutputDeleteSilenceOnce sync.Once
var exampleOutputDeleteSilence map[string]any

var exampleOutputCreateAnnotationOnce sync.Once
var exampleOutputCreateAnnotation map[string]any

var exampleOutputListAnnotationsOnce sync.Once
var exampleOutputListAnnotations map[string]any

var exampleOutputDeleteAnnotationOnce sync.Once
var exampleOutputDeleteAnnotation map[string]any

var exampleOutputCreateHTTPSyntheticCheckOnce sync.Once
var exampleOutputCreateHTTPSyntheticCheck map[string]any

var exampleOutputGetHTTPSyntheticCheckOnce sync.Once
var exampleOutputGetHTTPSyntheticCheck map[string]any

var exampleOutputUpdateHTTPSyntheticCheckOnce sync.Once
var exampleOutputUpdateHTTPSyntheticCheck map[string]any

var exampleOutputDeleteHTTPSyntheticCheckOnce sync.Once
var exampleOutputDeleteHTTPSyntheticCheck map[string]any

func (q *QueryDataSource) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputQueryDataSourceOnce, exampleOutputQueryDataSourceBytes, &exampleOutputQueryDataSource)
}

func (c *CreateAlertRule) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputCreateAlertRuleOnce,
		exampleOutputCreateAlertRuleBytes,
		&exampleOutputCreateAlertRule,
	)
}

func (c *GetAlertRule) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputGetAlertRuleOnce,
		exampleOutputGetAlertRuleBytes,
		&exampleOutputGetAlertRule,
	)
}

func (c *UpdateAlertRule) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputUpdateAlertRuleOnce,
		exampleOutputUpdateAlertRuleBytes,
		&exampleOutputUpdateAlertRule,
	)
}

func (c *DeleteAlertRule) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputDeleteAlertRuleOnce,
		exampleOutputDeleteAlertRuleBytes,
		&exampleOutputDeleteAlertRule,
	)
}

func (c *ListAlertRules) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputListAlertRulesOnce,
		exampleOutputListAlertRulesBytes,
		&exampleOutputListAlertRules,
	)
}

func (t *OnAlertFiring) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnAlertFiringOnce, exampleDataOnAlertFiringBytes, &exampleDataOnAlertFiring)
}

func (c *GetDashboard) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputGetDashboardOnce,
		exampleOutputGetDashboardBytes,
		&exampleOutputGetDashboard,
	)
}

func (c *RenderPanel) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputRenderPanelOnce,
		exampleOutputRenderPanelBytes,
		&exampleOutputRenderPanel,
	)
}

func (l *ListSilences) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputListSilencesOnce, exampleOutputListSilencesBytes, &exampleOutputListSilences)
}

func (g *GetSilence) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetSilenceOnce, exampleOutputGetSilenceBytes, &exampleOutputGetSilence)
}

func (c *CreateSilence) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateSilenceOnce, exampleOutputCreateSilenceBytes, &exampleOutputCreateSilence)
}

func (d *DeleteSilence) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputDeleteSilenceOnce, exampleOutputDeleteSilenceBytes, &exampleOutputDeleteSilence)
}

func (c *CreateAnnotation) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateAnnotationOnce, exampleOutputCreateAnnotationBytes, &exampleOutputCreateAnnotation)
}

func (l *ListAnnotations) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputListAnnotationsOnce, exampleOutputListAnnotationsBytes, &exampleOutputListAnnotations)
}

func (d *DeleteAnnotation) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputDeleteAnnotationOnce, exampleOutputDeleteAnnotationBytes, &exampleOutputDeleteAnnotation)
}

func (c *CreateHTTPSyntheticCheck) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputCreateHTTPSyntheticCheckOnce,
		exampleOutputCreateHTTPSyntheticCheckBytes,
		&exampleOutputCreateHTTPSyntheticCheck,
	)
}

func (g *GetHTTPSyntheticCheck) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputGetHTTPSyntheticCheckOnce,
		exampleOutputGetHTTPSyntheticCheckBytes,
		&exampleOutputGetHTTPSyntheticCheck,
	)
}

func (c *UpdateHTTPSyntheticCheck) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputUpdateHTTPSyntheticCheckOnce,
		exampleOutputUpdateHTTPSyntheticCheckBytes,
		&exampleOutputUpdateHTTPSyntheticCheck,
	)
}

func (d *DeleteHTTPSyntheticCheck) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputDeleteHTTPSyntheticCheckOnce,
		exampleOutputDeleteHTTPSyntheticCheckBytes,
		&exampleOutputDeleteHTTPSyntheticCheck,
	)
}
