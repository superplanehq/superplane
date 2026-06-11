package prometheus

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_query.json
var exampleOutputQueryBytes []byte

//go:embed example_output_query_range.json
var exampleOutputQueryRangeBytes []byte

var (
	exampleOutputQueryOnce sync.Once
	exampleOutputQuery     map[string]any

	exampleOutputQueryRangeOnce sync.Once
	exampleOutputQueryRange     map[string]any
)

func (q *Query) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputQueryOnce, exampleOutputQueryBytes, &exampleOutputQuery)
}

func (q *QueryRange) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputQueryRangeOnce, exampleOutputQueryRangeBytes, &exampleOutputQueryRange)
}
