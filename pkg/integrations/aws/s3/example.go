package s3

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_create_bucket.json
var exampleOutputCreateBucketBytes []byte

var exampleOutputCreateBucketOnce sync.Once
var exampleOutputCreateBucket map[string]any

//go:embed example_output_get_bucket.json
var exampleOutputGetBucketBytes []byte

var exampleOutputGetBucketOnce sync.Once
var exampleOutputGetBucket map[string]any

//go:embed example_output_delete_bucket.json
var exampleOutputDeleteBucketBytes []byte

var exampleOutputDeleteBucketOnce sync.Once
var exampleOutputDeleteBucket map[string]any

func (c *CreateBucket) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateBucketOnce, exampleOutputCreateBucketBytes, &exampleOutputCreateBucket)
}

func (c *GetBucket) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetBucketOnce, exampleOutputGetBucketBytes, &exampleOutputGetBucket)
}

func (c *DeleteBucket) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputDeleteBucketOnce, exampleOutputDeleteBucketBytes, &exampleOutputDeleteBucket)
}
