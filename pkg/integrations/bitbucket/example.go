package bitbucket

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_data_on_push.json
var exampleDataOnPushBytes []byte

var exampleDataOnPushOnce sync.Once
var exampleDataOnPush map[string]any

func (t *OnPush) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnPushOnce, exampleDataOnPushBytes, &exampleDataOnPush)
}
