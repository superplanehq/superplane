package ecr

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_data_on_image_push.json
var exampleDataOnImagePushBytes []byte

var exampleDataOnImagePushOnce sync.Once
var exampleDataOnImagePush map[string]any

func (t *OnImagePush) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnImagePushOnce, exampleDataOnImagePushBytes, &exampleDataOnImagePush)
}
