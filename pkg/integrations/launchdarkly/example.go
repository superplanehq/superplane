package launchdarkly

import (
	_ "embed"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_get_feature_flag.json
var exampleOutputGetFeatureFlagBytes []byte
var exampleOutputGetFeatureFlag = utils.NewEmbeddedJSON(exampleOutputGetFeatureFlagBytes)

//go:embed example_output_delete_feature_flag.json
var exampleOutputDeleteFeatureFlagBytes []byte
var exampleOutputDeleteFeatureFlag = utils.NewEmbeddedJSON(exampleOutputDeleteFeatureFlagBytes)

//go:embed example_data_on_feature_flag_change.json
var exampleDataOnFeatureFlagChangeBytes []byte
var exampleDataOnFeatureFlagChange = utils.NewEmbeddedJSON(exampleDataOnFeatureFlagChangeBytes)

func (c *GetFeatureFlag) ExampleOutput() map[string]any {
	return exampleOutputGetFeatureFlag.Value()
}

func (c *DeleteFeatureFlag) ExampleOutput() map[string]any {
	return exampleOutputDeleteFeatureFlag.Value()
}

func (t *OnFeatureFlagChange) ExampleData() map[string]any {
	return exampleDataOnFeatureFlagChange.Value()
}
