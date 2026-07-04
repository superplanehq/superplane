package monitoring

import (
	_ "embed"
	"sync"

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
	exampleOutputCreateAlertingPolicyOnce sync.Once
	exampleOutputCreateAlertingPolicy     map[string]any

	exampleOutputGetAlertingPolicyOnce sync.Once
	exampleOutputGetAlertingPolicy     map[string]any

	exampleOutputDeleteAlertingPolicyOnce sync.Once
	exampleOutputDeleteAlertingPolicy     map[string]any

	exampleOutputUpdateAlertingPolicyOnce sync.Once
	exampleOutputUpdateAlertingPolicy     map[string]any

	exampleOutputCreateSnoozeOnce sync.Once
	exampleOutputCreateSnooze     map[string]any

	exampleOutputGetSnoozeOnce sync.Once
	exampleOutputGetSnooze     map[string]any

	exampleOutputExpireSnoozeOnce sync.Once
	exampleOutputExpireSnooze     map[string]any

	exampleDataOnAlertOnce sync.Once
	exampleDataOnAlert     map[string]any
)

func (c *CreateSnooze) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateSnoozeOnce, exampleOutputCreateSnoozeBytes, &exampleOutputCreateSnooze)
}

func (g *GetSnooze) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetSnoozeOnce, exampleOutputGetSnoozeBytes, &exampleOutputGetSnooze)
}

func (e *ExpireSnooze) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputExpireSnoozeOnce, exampleOutputExpireSnoozeBytes, &exampleOutputExpireSnooze)
}

func onAlertExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnAlertOnce, exampleDataOnAlertBytes, &exampleDataOnAlert)
}

func (c *CreateAlertingPolicy) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateAlertingPolicyOnce, exampleOutputCreateAlertingPolicyBytes, &exampleOutputCreateAlertingPolicy)
}

func (g *GetAlertingPolicy) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetAlertingPolicyOnce, exampleOutputGetAlertingPolicyBytes, &exampleOutputGetAlertingPolicy)
}

func (d *DeleteAlertingPolicy) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputDeleteAlertingPolicyOnce, exampleOutputDeleteAlertingPolicyBytes, &exampleOutputDeleteAlertingPolicy)
}

func (u *UpdateAlertingPolicy) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputUpdateAlertingPolicyOnce, exampleOutputUpdateAlertingPolicyBytes, &exampleOutputUpdateAlertingPolicy)
}
