package hetzner

import (
	_ "embed"

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

//go:embed example_output_delete_object.json
var exampleOutputDeleteObjectBytes []byte

//go:embed example_output_list_objects.json
var exampleOutputListObjectsBytes []byte

//go:embed example_output_presigned_url.json
var exampleOutputPresignedURLBytes []byte

var (
	exampleOutputCreateServer       = utils.NewEmbeddedJSON(exampleOutputCreateServerBytes)
	exampleOutputCreateSnapshot     = utils.NewEmbeddedJSON(exampleOutputCreateSnapshotBytes)
	exampleOutputCreateLoadBalancer = utils.NewEmbeddedJSON(exampleOutputCreateLoadBalancerBytes)
	exampleOutputDeleteServer       = utils.NewEmbeddedJSON(exampleOutputDeleteServerBytes)
	exampleOutputDeleteSnapshot     = utils.NewEmbeddedJSON(exampleOutputDeleteSnapshotBytes)
	exampleOutputDeleteLoadBalancer = utils.NewEmbeddedJSON(exampleOutputDeleteLoadBalancerBytes)
	exampleOutputCreateBucketData   = utils.NewEmbeddedJSON(exampleOutputCreateBucketBytes)
	exampleOutputDeleteBucketData   = utils.NewEmbeddedJSON(exampleOutputDeleteBucketBytes)
	exampleOutputUploadObjectData   = utils.NewEmbeddedJSON(exampleOutputUploadObjectBytes)
	exampleOutputDeleteObjectData   = utils.NewEmbeddedJSON(exampleOutputDeleteObjectBytes)
	exampleOutputListObjectsData    = utils.NewEmbeddedJSON(exampleOutputListObjectsBytes)
	exampleOutputPresignedURLData   = utils.NewEmbeddedJSON(exampleOutputPresignedURLBytes)
)

func (c *CreateServer) ExampleOutput() map[string]any {
	return exampleOutputCreateServer.Value()
}

func (c *CreateSnapshot) ExampleOutput() map[string]any {
	return exampleOutputCreateSnapshot.Value()
}

func (c *CreateLoadBalancer) ExampleOutput() map[string]any {
	return exampleOutputCreateLoadBalancer.Value()
}

func (c *DeleteServer) ExampleOutput() map[string]any {
	return exampleOutputDeleteServer.Value()
}

func (c *DeleteSnapshot) ExampleOutput() map[string]any {
	return exampleOutputDeleteSnapshot.Value()
}

func (c *DeleteLoadBalancer) ExampleOutput() map[string]any {
	return exampleOutputDeleteLoadBalancer.Value()
}

func exampleOutputCreateBucket() map[string]any {
	return exampleOutputCreateBucketData.Value()
}

func exampleOutputDeleteBucket() map[string]any {
	return exampleOutputDeleteBucketData.Value()
}

func exampleOutputUploadObject() map[string]any {
	return exampleOutputUploadObjectData.Value()
}

func exampleOutputDeleteObject() map[string]any {
	return exampleOutputDeleteObjectData.Value()
}

func exampleOutputListObjects() map[string]any {
	return exampleOutputListObjectsData.Value()
}

func exampleOutputPresignedURL() map[string]any {
	return exampleOutputPresignedURLData.Value()
}
