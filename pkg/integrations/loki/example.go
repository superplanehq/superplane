package loki

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_push_logs.json
var exampleOutputPushLogsBytes []byte

var exampleOutputPushLogsOnce sync.Once
var exampleOutputPushLogsData map[string]any

func exampleOutputPushLogs() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputPushLogsOnce, exampleOutputPushLogsBytes, &exampleOutputPushLogsData)
}

//go:embed example_output_query_logs.json
var exampleOutputQueryLogsBytes []byte

var exampleOutputQueryLogsOnce sync.Once
var exampleOutputQueryLogsData map[string]any

func exampleOutputQueryLogs() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputQueryLogsOnce, exampleOutputQueryLogsBytes, &exampleOutputQueryLogsData)
}
