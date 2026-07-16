package bitbucket

import (
	_ "embed"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_data_on_push.json
var exampleDataOnPushBytes []byte
var exampleDataOnPush = utils.NewEmbeddedJSON(exampleDataOnPushBytes)

func (t *OnPush) ExampleData() map[string]any {
	return exampleDataOnPush.Value()
}
