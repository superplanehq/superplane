package storage

import (
	_ "embed"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_create_bucket.json
var exampleOutputCreateBucketBytes []byte

//go:embed example_output_get_bucket.json
var exampleOutputGetBucketBytes []byte

//go:embed example_output_delete_bucket.json
var exampleOutputDeleteBucketBytes []byte

var (
	exampleOutputCreateBucket = utils.NewEmbeddedJSON(exampleOutputCreateBucketBytes)
	exampleOutputGetBucket    = utils.NewEmbeddedJSON(exampleOutputGetBucketBytes)
	exampleOutputDeleteBucket = utils.NewEmbeddedJSON(exampleOutputDeleteBucketBytes)
)

func (c *CreateBucket) ExampleOutput() map[string]any {
	return exampleOutputCreateBucket.Value()
}

func (g *GetBucket) ExampleOutput() map[string]any {
	return exampleOutputGetBucket.Value()
}

func (d *DeleteBucket) ExampleOutput() map[string]any {
	return exampleOutputDeleteBucket.Value()
}
