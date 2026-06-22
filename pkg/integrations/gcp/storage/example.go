package storage

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_create_bucket.json
var exampleOutputCreateBucketBytes []byte

//go:embed example_output_get_bucket.json
var exampleOutputGetBucketBytes []byte

//go:embed example_output_delete_bucket.json
var exampleOutputDeleteBucketBytes []byte

var (
	exampleOutputCreateBucketOnce sync.Once
	exampleOutputCreateBucket     map[string]any

	exampleOutputGetBucketOnce sync.Once
	exampleOutputGetBucket     map[string]any

	exampleOutputDeleteBucketOnce sync.Once
	exampleOutputDeleteBucket     map[string]any
)

func (c *CreateBucket) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateBucketOnce, exampleOutputCreateBucketBytes, &exampleOutputCreateBucket)
}

func (g *GetBucket) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetBucketOnce, exampleOutputGetBucketBytes, &exampleOutputGetBucket)
}

func (d *DeleteBucket) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputDeleteBucketOnce, exampleOutputDeleteBucketBytes, &exampleOutputDeleteBucket)
}
