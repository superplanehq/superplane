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

func (c *CreateBucket) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateBucketOnce, exampleOutputCreateBucketBytes, &exampleOutputCreateBucket)
}
