package productive

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_data_on_task.json
var exampleDataOnTaskBytes []byte

var exampleDataOnTaskOnce sync.Once
var exampleDataOnTask map[string]any

func (t *OnTask) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnTaskOnce, exampleDataOnTaskBytes, &exampleDataOnTask)
}
