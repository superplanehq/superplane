package launchdarkly

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_get_feature_flag.json
var exampleOutputGetFeatureFlagBytes []byte

var exampleOutputGetFeatureFlagOnce sync.Once
var exampleOutputGetFeatureFlag map[string]any

//go:embed example_output_delete_feature_flag.json
var exampleOutputDeleteFeatureFlagBytes []byte

var exampleOutputDeleteFeatureFlagOnce sync.Once
var exampleOutputDeleteFeatureFlag map[string]any

//go:embed example_data_on_feature_flag_change.json
var exampleDataOnFeatureFlagChangeBytes []byte

var exampleDataOnFeatureFlagChangeOnce sync.Once
var exampleDataOnFeatureFlagChange map[string]any

func (c *GetFeatureFlag) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetFeatureFlagOnce, exampleOutputGetFeatureFlagBytes, &exampleOutputGetFeatureFlag)
}

func (c *DeleteFeatureFlag) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputDeleteFeatureFlagOnce, exampleOutputDeleteFeatureFlagBytes, &exampleOutputDeleteFeatureFlag)
}

func (t *OnFeatureFlagChange) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnFeatureFlagChangeOnce, exampleDataOnFeatureFlagChangeBytes, &exampleDataOnFeatureFlagChange)
}
