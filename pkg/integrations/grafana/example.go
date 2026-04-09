package grafana

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_query_data_source.json
var exampleOutputQueryDataSourceBytes []byte

//go:embed example_data_on_alert_firing.json
var exampleDataOnAlertFiringBytes []byte

//go:embed example_output_get_dashboard.json
var exampleOutputGetDashboardBytes []byte

//go:embed example_output_render_panel.json
var exampleOutputRenderPanelBytes []byte

var exampleOutputQueryDataSourceOnce sync.Once
var exampleOutputQueryDataSource map[string]any

var exampleDataOnAlertFiringOnce sync.Once
var exampleDataOnAlertFiring map[string]any

var exampleOutputGetDashboardOnce sync.Once
var exampleOutputGetDashboard map[string]any

var exampleOutputRenderPanelOnce sync.Once
var exampleOutputRenderPanel map[string]any

func (q *QueryDataSource) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputQueryDataSourceOnce, exampleOutputQueryDataSourceBytes, &exampleOutputQueryDataSource)
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
