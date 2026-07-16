package grafana

import (
	_ "embed"

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

//go:embed example_output_query_logs.json
var exampleOutputQueryLogsBytes []byte

//go:embed example_output_query_traces.json
var exampleOutputQueryTracesBytes []byte

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

//go:embed example_output_declare_incident.json
var exampleOutputDeclareIncidentBytes []byte

//go:embed example_output_declare_drill.json
var exampleOutputDeclareDrillBytes []byte

//go:embed example_output_get_incident.json
var exampleOutputGetIncidentBytes []byte

//go:embed example_output_update_incident.json
var exampleOutputUpdateIncidentBytes []byte

//go:embed example_output_resolve_incident.json
var exampleOutputResolveIncidentBytes []byte

//go:embed example_output_add_incident_activity.json
var exampleOutputAddIncidentActivityBytes []byte

//go:embed example_output_create_http_synthetic_check.json
var exampleOutputCreateHTTPSyntheticCheckBytes []byte

//go:embed example_output_get_http_synthetic_check.json
var exampleOutputGetHTTPSyntheticCheckBytes []byte

//go:embed example_output_update_http_synthetic_check.json
var exampleOutputUpdateHTTPSyntheticCheckBytes []byte

//go:embed example_output_delete_http_synthetic_check.json
var exampleOutputDeleteHTTPSyntheticCheckBytes []byte
var exampleOutputQueryDataSource = utils.NewEmbeddedJSON(exampleOutputQueryDataSourceBytes)
var exampleOutputCreateAlertRule = utils.NewEmbeddedJSON(exampleOutputCreateAlertRuleBytes)
var exampleOutputGetAlertRule = utils.NewEmbeddedJSON(exampleOutputGetAlertRuleBytes)
var exampleOutputUpdateAlertRule = utils.NewEmbeddedJSON(exampleOutputUpdateAlertRuleBytes)
var exampleOutputDeleteAlertRule = utils.NewEmbeddedJSON(exampleOutputDeleteAlertRuleBytes)
var exampleOutputListAlertRules = utils.NewEmbeddedJSON(exampleOutputListAlertRulesBytes)
var exampleDataOnAlertFiring = utils.NewEmbeddedJSON(exampleDataOnAlertFiringBytes)
var exampleOutputQueryLogs = utils.NewEmbeddedJSON(exampleOutputQueryLogsBytes)
var exampleOutputQueryTraces = utils.NewEmbeddedJSON(exampleOutputQueryTracesBytes)
var exampleOutputGetDashboard = utils.NewEmbeddedJSON(exampleOutputGetDashboardBytes)
var exampleOutputRenderPanel = utils.NewEmbeddedJSON(exampleOutputRenderPanelBytes)
var exampleOutputListSilences = utils.NewEmbeddedJSON(exampleOutputListSilencesBytes)
var exampleOutputGetSilence = utils.NewEmbeddedJSON(exampleOutputGetSilenceBytes)
var exampleOutputCreateSilence = utils.NewEmbeddedJSON(exampleOutputCreateSilenceBytes)
var exampleOutputDeleteSilence = utils.NewEmbeddedJSON(exampleOutputDeleteSilenceBytes)
var exampleOutputCreateAnnotation = utils.NewEmbeddedJSON(exampleOutputCreateAnnotationBytes)
var exampleOutputListAnnotations = utils.NewEmbeddedJSON(exampleOutputListAnnotationsBytes)
var exampleOutputDeleteAnnotation = utils.NewEmbeddedJSON(exampleOutputDeleteAnnotationBytes)
var exampleOutputDeclareIncident = utils.NewEmbeddedJSON(exampleOutputDeclareIncidentBytes)
var exampleOutputDeclareDrill = utils.NewEmbeddedJSON(exampleOutputDeclareDrillBytes)
var exampleOutputGetIncident = utils.NewEmbeddedJSON(exampleOutputGetIncidentBytes)
var exampleOutputUpdateIncident = utils.NewEmbeddedJSON(exampleOutputUpdateIncidentBytes)
var exampleOutputResolveIncident = utils.NewEmbeddedJSON(exampleOutputResolveIncidentBytes)
var exampleOutputAddIncidentActivity = utils.NewEmbeddedJSON(exampleOutputAddIncidentActivityBytes)
var exampleOutputCreateHTTPSyntheticCheck = utils.NewEmbeddedJSON(exampleOutputCreateHTTPSyntheticCheckBytes)
var exampleOutputGetHTTPSyntheticCheck = utils.NewEmbeddedJSON(exampleOutputGetHTTPSyntheticCheckBytes)
var exampleOutputUpdateHTTPSyntheticCheck = utils.NewEmbeddedJSON(exampleOutputUpdateHTTPSyntheticCheckBytes)
var exampleOutputDeleteHTTPSyntheticCheck = utils.NewEmbeddedJSON(exampleOutputDeleteHTTPSyntheticCheckBytes)

func (q *QueryDataSource) ExampleOutput() map[string]any {
	return exampleOutputQueryDataSource.Value()
}

func (c *CreateAlertRule) ExampleOutput() map[string]any {
	return exampleOutputCreateAlertRule.Value()
}

func (c *GetAlertRule) ExampleOutput() map[string]any {
	return exampleOutputGetAlertRule.Value()
}

func (c *UpdateAlertRule) ExampleOutput() map[string]any {
	return exampleOutputUpdateAlertRule.Value()
}

func (c *DeleteAlertRule) ExampleOutput() map[string]any {
	return exampleOutputDeleteAlertRule.Value()
}

func (c *ListAlertRules) ExampleOutput() map[string]any {
	return exampleOutputListAlertRules.Value()
}

func (t *OnAlertFiring) ExampleData() map[string]any {
	return exampleDataOnAlertFiring.Value()
}

func (q *QueryLogs) ExampleOutput() map[string]any {
	return exampleOutputQueryLogs.Value()
}

func (q *QueryTraces) ExampleOutput() map[string]any {
	return exampleOutputQueryTraces.Value()
}

func (c *GetDashboard) ExampleOutput() map[string]any {
	return exampleOutputGetDashboard.Value()
}

func (c *RenderPanel) ExampleOutput() map[string]any {
	return exampleOutputRenderPanel.Value()
}

func (l *ListSilences) ExampleOutput() map[string]any {
	return exampleOutputListSilences.Value()
}

func (g *GetSilence) ExampleOutput() map[string]any {
	return exampleOutputGetSilence.Value()
}

func (c *CreateSilence) ExampleOutput() map[string]any {
	return exampleOutputCreateSilence.Value()
}

func (d *DeleteSilence) ExampleOutput() map[string]any {
	return exampleOutputDeleteSilence.Value()
}

func (c *CreateAnnotation) ExampleOutput() map[string]any {
	return exampleOutputCreateAnnotation.Value()
}

func (l *ListAnnotations) ExampleOutput() map[string]any {
	return exampleOutputListAnnotations.Value()
}

func (d *DeleteAnnotation) ExampleOutput() map[string]any {
	return exampleOutputDeleteAnnotation.Value()
}

func (d *DeclareIncident) ExampleOutput() map[string]any {
	return exampleOutputDeclareIncident.Value()
}

func (d *DeclareDrill) ExampleOutput() map[string]any {
	return exampleOutputDeclareDrill.Value()
}

func (g *GetIncident) ExampleOutput() map[string]any {
	return exampleOutputGetIncident.Value()
}

func (u *UpdateIncident) ExampleOutput() map[string]any {
	return exampleOutputUpdateIncident.Value()
}

func (r *ResolveIncident) ExampleOutput() map[string]any {
	return exampleOutputResolveIncident.Value()
}

func (a *AddIncidentActivity) ExampleOutput() map[string]any {
	return exampleOutputAddIncidentActivity.Value()
}

func (c *CreateHTTPSyntheticCheck) ExampleOutput() map[string]any {
	return exampleOutputCreateHTTPSyntheticCheck.Value()
}

func (g *GetHTTPSyntheticCheck) ExampleOutput() map[string]any {
	return exampleOutputGetHTTPSyntheticCheck.Value()
}

func (c *UpdateHTTPSyntheticCheck) ExampleOutput() map[string]any {
	return exampleOutputUpdateHTTPSyntheticCheck.Value()
}

func (d *DeleteHTTPSyntheticCheck) ExampleOutput() map[string]any {
	return exampleOutputDeleteHTTPSyntheticCheck.Value()
}
