package launchdarkly

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_get_flag.json
var exampleOutputGetFlagBytes []byte

var exampleOutputGetFlagOnce sync.Once
var exampleOutputGetFlag map[string]any

//go:embed example_output_delete_flag.json
var exampleOutputDeleteFlagBytes []byte

var exampleOutputDeleteFlagOnce sync.Once
var exampleOutputDeleteFlag map[string]any

func getFlagExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetFlagOnce, exampleOutputGetFlagBytes, &exampleOutputGetFlag)
}

func deleteFlagExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputDeleteFlagOnce, exampleOutputDeleteFlagBytes, &exampleOutputDeleteFlag)
}
