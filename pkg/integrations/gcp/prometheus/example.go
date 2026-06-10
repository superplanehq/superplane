package prometheus

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_query.json
var exampleOutputQueryBytes []byte

var (
	exampleOutputQueryOnce sync.Once
	exampleOutputQuery     map[string]any
)

func (q *Query) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputQueryOnce, exampleOutputQueryBytes, &exampleOutputQuery)
}
