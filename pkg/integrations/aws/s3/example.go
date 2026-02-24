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

//go:embed example_output_delete_bucket.json
var exampleOutputDeleteBucketBytes []byte

var exampleOutputDeleteBucketOnce sync.Once
var exampleOutputDeleteBucket map[string]any

//go:embed example_output_head_bucket.json
var exampleOutputHeadBucketBytes []byte

var exampleOutputHeadBucketOnce sync.Once
var exampleOutputHeadBucket map[string]any

//go:embed example_output_empty_bucket.json
var exampleOutputEmptyBucketBytes []byte

var exampleOutputEmptyBucketOnce sync.Once
var exampleOutputEmptyBucket map[string]any

//go:embed example_output_copy_object.json
var exampleOutputCopyObjectBytes []byte

var exampleOutputCopyObjectOnce sync.Once
var exampleOutputCopyObject map[string]any

//go:embed example_output_delete_object.json
var exampleOutputDeleteObjectBytes []byte

var exampleOutputDeleteObjectOnce sync.Once
var exampleOutputDeleteObject map[string]any

//go:embed example_output_head_object.json
var exampleOutputHeadObjectBytes []byte

var exampleOutputHeadObjectOnce sync.Once
var exampleOutputHeadObject map[string]any

//go:embed example_output_get_object_attributes.json
var exampleOutputGetObjectAttributesBytes []byte

var exampleOutputGetObjectAttributesOnce sync.Once
var exampleOutputGetObjectAttributes map[string]any

//go:embed example_output_put_object.json
var exampleOutputPutObjectBytes []byte

var exampleOutputPutObjectOnce sync.Once
var exampleOutputPutObject map[string]any

func (c *CreateBucket) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateBucketOnce, exampleOutputCreateBucketBytes, &exampleOutputCreateBucket)
}

func (c *DeleteBucket) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputDeleteBucketOnce, exampleOutputDeleteBucketBytes, &exampleOutputDeleteBucket)
}

func (c *HeadBucket) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputHeadBucketOnce, exampleOutputHeadBucketBytes, &exampleOutputHeadBucket)
}

func (c *EmptyBucket) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputEmptyBucketOnce, exampleOutputEmptyBucketBytes, &exampleOutputEmptyBucket)
}

func (c *CopyObject) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCopyObjectOnce, exampleOutputCopyObjectBytes, &exampleOutputCopyObject)
}

func (c *DeleteObject) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputDeleteObjectOnce, exampleOutputDeleteObjectBytes, &exampleOutputDeleteObject)
}

func (c *HeadObject) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputHeadObjectOnce, exampleOutputHeadObjectBytes, &exampleOutputHeadObject)
}

func (c *GetObjectAttributes) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetObjectAttributesOnce, exampleOutputGetObjectAttributesBytes, &exampleOutputGetObjectAttributes)
}

func (c *PutObject) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputPutObjectOnce, exampleOutputPutObjectBytes, &exampleOutputPutObject)
}
