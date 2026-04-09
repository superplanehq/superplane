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
