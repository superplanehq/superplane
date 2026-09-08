package notion

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_data_on_page_added.json
var exampleDataOnPageAddedBytes []byte

var exampleDataOnPageAddedOnce sync.Once
var exampleDataOnPageAdded map[string]any

func (t *OnPageAdded) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnPageAddedOnce, exampleDataOnPageAddedBytes, &exampleDataOnPageAdded)
}
