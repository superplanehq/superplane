package onerror

import (
	_ "embed"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_data.json
var exampleDataBytes []byte
var exampleData = utils.NewEmbeddedJSON(exampleDataBytes)

func (t *OnError) ExampleData() map[string]any {
	return exampleData.Value()
}
