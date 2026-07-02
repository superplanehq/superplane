package onerror

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_data.json
var exampleDataBytes []byte

var exampleDataOnce sync.Once
var exampleData map[string]any

func (t *OnError) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnce, exampleDataBytes, &exampleData)
}
