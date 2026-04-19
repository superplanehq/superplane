package hetzner

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_create_server.json
var exampleOutputCreateServerBytes []byte

//go:embed example_output_create_snapshot.json
var exampleOutputCreateSnapshotBytes []byte

//go:embed example_output_create_load_balancer.json
var exampleOutputCreateLoadBalancerBytes []byte

//go:embed example_output_delete_server.json
var exampleOutputDeleteServerBytes []byte

//go:embed example_output_delete_snapshot.json
var exampleOutputDeleteSnapshotBytes []byte

//go:embed example_output_delete_load_balancer.json
var exampleOutputDeleteLoadBalancerBytes []byte

//go:embed example_output_create_bucket.json
var exampleOutputCreateBucketBytes []byte

//go:embed example_output_delete_bucket.json
var exampleOutputDeleteBucketBytes []byte

//go:embed example_output_upload_object.json
var exampleOutputUploadObjectBytes []byte

//go:embed example_output_download_object.json
var exampleOutputDownloadObjectBytes []byte

//go:embed example_output_delete_object.json
var exampleOutputDeleteObjectBytes []byte

//go:embed example_output_list_objects.json
var exampleOutputListObjectsBytes []byte

//go:embed example_output_presigned_url.json
var exampleOutputPresignedURLBytes []byte

var (
	exampleOutputCreateServerOnce sync.Once
	exampleOutputCreateServer     map[string]any

	exampleOutputCreateSnapshotOnce sync.Once
	exampleOutputCreateSnapshot     map[string]any

	exampleOutputCreateLoadBalancerOnce sync.Once
	exampleOutputCreateLoadBalancer     map[string]any

	exampleOutputDeleteServerOnce sync.Once
	exampleOutputDeleteServer     map[string]any

	exampleOutputDeleteSnapshotOnce sync.Once
	exampleOutputDeleteSnapshot     map[string]any

	exampleOutputDeleteLoadBalancerOnce sync.Once
	exampleOutputDeleteLoadBalancer     map[string]any

	exampleOutputCreateBucketOnce sync.Once
	exampleOutputCreateBucketData map[string]any

	exampleOutputDeleteBucketOnce sync.Once
	exampleOutputDeleteBucketData map[string]any

	exampleOutputUploadObjectOnce sync.Once
	exampleOutputUploadObjectData map[string]any

	exampleOutputDownloadObjectOnce sync.Once
	exampleOutputDownloadObjectData map[string]any

	exampleOutputDeleteObjectOnce sync.Once
	exampleOutputDeleteObjectData map[string]any

	exampleOutputListObjectsOnce sync.Once
	exampleOutputListObjectsData map[string]any

	exampleOutputPresignedURLOnce sync.Once
	exampleOutputPresignedURLData map[string]any
)

func (c *CreateServer) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateServerOnce, exampleOutputCreateServerBytes, &exampleOutputCreateServer)
}

func (c *CreateSnapshot) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateSnapshotOnce, exampleOutputCreateSnapshotBytes, &exampleOutputCreateSnapshot)
}

func (c *CreateLoadBalancer) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateLoadBalancerOnce, exampleOutputCreateLoadBalancerBytes, &exampleOutputCreateLoadBalancer)
}

func (c *DeleteServer) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputDeleteServerOnce, exampleOutputDeleteServerBytes, &exampleOutputDeleteServer)
}

func (c *DeleteSnapshot) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputDeleteSnapshotOnce, exampleOutputDeleteSnapshotBytes, &exampleOutputDeleteSnapshot)
}

func (c *DeleteLoadBalancer) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputDeleteLoadBalancerOnce, exampleOutputDeleteLoadBalancerBytes, &exampleOutputDeleteLoadBalancer)
}

func exampleOutputCreateBucket() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateBucketOnce, exampleOutputCreateBucketBytes, &exampleOutputCreateBucketData)
}

func exampleOutputDeleteBucket() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputDeleteBucketOnce, exampleOutputDeleteBucketBytes, &exampleOutputDeleteBucketData)
}

func exampleOutputUploadObject() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputUploadObjectOnce, exampleOutputUploadObjectBytes, &exampleOutputUploadObjectData)
}

func exampleOutputDownloadObject() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputDownloadObjectOnce, exampleOutputDownloadObjectBytes, &exampleOutputDownloadObjectData)
}

func exampleOutputDeleteObject() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputDeleteObjectOnce, exampleOutputDeleteObjectBytes, &exampleOutputDeleteObjectData)
}

func exampleOutputListObjects() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputListObjectsOnce, exampleOutputListObjectsBytes, &exampleOutputListObjectsData)
}

func exampleOutputPresignedURL() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputPresignedURLOnce, exampleOutputPresignedURLBytes, &exampleOutputPresignedURLData)
}
