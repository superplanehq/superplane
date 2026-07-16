package monitoring

import (
	_ "embed"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_create_alerting_policy.json
var exampleOutputCreateAlertingPolicyBytes []byte

//go:embed example_output_get_alerting_policy.json
var exampleOutputGetAlertingPolicyBytes []byte

//go:embed example_output_delete_alerting_policy.json
var exampleOutputDeleteAlertingPolicyBytes []byte

//go:embed example_output_update_alerting_policy.json
var exampleOutputUpdateAlertingPolicyBytes []byte

//go:embed example_output_create_snooze.json
var exampleOutputCreateSnoozeBytes []byte

//go:embed example_output_get_snooze.json
var exampleOutputGetSnoozeBytes []byte

//go:embed example_output_expire_snooze.json
var exampleOutputExpireSnoozeBytes []byte

//go:embed example_data_on_alert.json
var exampleDataOnAlertBytes []byte

var (
	exampleOutputCreateAlertingPolicy = utils.NewEmbeddedJSON(exampleOutputCreateAlertingPolicyBytes)
	exampleOutputGetAlertingPolicy    = utils.NewEmbeddedJSON(exampleOutputGetAlertingPolicyBytes)
	exampleOutputDeleteAlertingPolicy = utils.NewEmbeddedJSON(exampleOutputDeleteAlertingPolicyBytes)
	exampleOutputUpdateAlertingPolicy = utils.NewEmbeddedJSON(exampleOutputUpdateAlertingPolicyBytes)
	exampleOutputCreateSnooze         = utils.NewEmbeddedJSON(exampleOutputCreateSnoozeBytes)
	exampleOutputGetSnooze            = utils.NewEmbeddedJSON(exampleOutputGetSnoozeBytes)
	exampleOutputExpireSnooze         = utils.NewEmbeddedJSON(exampleOutputExpireSnoozeBytes)
	exampleDataOnAlert                = utils.NewEmbeddedJSON(exampleDataOnAlertBytes)
)

func (c *CreateSnooze) ExampleOutput() map[string]any {
	return exampleOutputCreateSnooze.Value()
}

func (g *GetSnooze) ExampleOutput() map[string]any {
	return exampleOutputGetSnooze.Value()
}

func (e *ExpireSnooze) ExampleOutput() map[string]any {
	return exampleOutputExpireSnooze.Value()
}

func onAlertExampleData() map[string]any {
	return exampleDataOnAlert.Value()
}

func (c *CreateAlertingPolicy) ExampleOutput() map[string]any {
	return exampleOutputCreateAlertingPolicy.Value()
}

func (g *GetAlertingPolicy) ExampleOutput() map[string]any {
	return exampleOutputGetAlertingPolicy.Value()
}

func (d *DeleteAlertingPolicy) ExampleOutput() map[string]any {
	return exampleOutputDeleteAlertingPolicy.Value()
}

func (u *UpdateAlertingPolicy) ExampleOutput() map[string]any {
	return exampleOutputUpdateAlertingPolicy.Value()
}
