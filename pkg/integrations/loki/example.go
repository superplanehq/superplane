package loki

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_push_logs.json
var exampleOutputPushLogsBytes []byte

var exampleOutputPushLogsOnce sync.Once
var exampleOutputPushLogs map[string]any

func (c *PushLogs) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputPushLogsOnce, exampleOutputPushLogsBytes, &exampleOutputPushLogs)
}

//go:embed example_output_query_logs.json
var exampleOutputQueryLogsBytes []byte

var exampleOutputQueryLogsOnce sync.Once
var exampleOutputQueryLogs map[string]any

func (c *QueryLogs) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputQueryLogsOnce, exampleOutputQueryLogsBytes, &exampleOutputQueryLogs)
}
