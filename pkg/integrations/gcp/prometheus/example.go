package prometheus

import (
	_ "embed"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_query.json
var exampleOutputQueryBytes []byte

//go:embed example_output_query_range.json
var exampleOutputQueryRangeBytes []byte

var (
	exampleOutputQuery      = utils.NewEmbeddedJSON(exampleOutputQueryBytes)
	exampleOutputQueryRange = utils.NewEmbeddedJSON(exampleOutputQueryRangeBytes)
)

func (q *Query) ExampleOutput() map[string]any {
	return exampleOutputQuery.Value()
}

func (q *QueryRange) ExampleOutput() map[string]any {
	return exampleOutputQueryRange.Value()
}
