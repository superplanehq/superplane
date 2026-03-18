package digitalocean

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_create_droplet.json
var exampleOutputCreateDropletBytes []byte

var exampleOutputCreateDropletOnce sync.Once
var exampleOutputCreateDroplet map[string]any

func (c *CreateDroplet) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateDropletOnce, exampleOutputCreateDropletBytes, &exampleOutputCreateDroplet)
}

//go:embed example_output_get_droplet.json
var exampleOutputGetDropletBytes []byte

var exampleOutputGetDropletOnce sync.Once
var exampleOutputGetDroplet map[string]any

func (g *GetDroplet) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetDropletOnce, exampleOutputGetDropletBytes, &exampleOutputGetDroplet)
}

//go:embed example_output_delete_droplet.json
var exampleOutputDeleteDropletBytes []byte

var exampleOutputDeleteDropletOnce sync.Once
var exampleOutputDeleteDroplet map[string]any

func (d *DeleteDroplet) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputDeleteDropletOnce, exampleOutputDeleteDropletBytes, &exampleOutputDeleteDroplet)
}

//go:embed example_output_manage_droplet_power.json
var exampleOutputManageDropletPowerBytes []byte

var exampleOutputManageDropletPowerOnce sync.Once
var exampleOutputManageDropletPower map[string]any

func (m *ManageDropletPower) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputManageDropletPowerOnce, exampleOutputManageDropletPowerBytes, &exampleOutputManageDropletPower)
}

//go:embed example_output_create_snapshot.json
var exampleOutputCreateSnapshotBytes []byte

var exampleOutputCreateSnapshotOnce sync.Once
var exampleOutputCreateSnapshot map[string]any

func (c *CreateSnapshot) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateSnapshotOnce, exampleOutputCreateSnapshotBytes, &exampleOutputCreateSnapshot)
}

//go:embed example_output_delete_snapshot.json
var exampleOutputDeleteSnapshotBytes []byte

var exampleOutputDeleteSnapshotOnce sync.Once
var exampleOutputDeleteSnapshot map[string]any

func (c *DeleteSnapshot) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputDeleteSnapshotOnce, exampleOutputDeleteSnapshotBytes, &exampleOutputDeleteSnapshot)
}

//go:embed example_output_create_dns_record.json
var exampleOutputCreateDNSRecordBytes []byte

var exampleOutputCreateDNSRecordOnce sync.Once
var exampleOutputCreateDNSRecord map[string]any

func (c *CreateDNSRecord) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateDNSRecordOnce, exampleOutputCreateDNSRecordBytes, &exampleOutputCreateDNSRecord)
}

//go:embed example_output_delete_dns_record.json
var exampleOutputDeleteDNSRecordBytes []byte

var exampleOutputDeleteDNSRecordOnce sync.Once
var exampleOutputDeleteDNSRecord map[string]any

func (d *DeleteDNSRecord) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputDeleteDNSRecordOnce, exampleOutputDeleteDNSRecordBytes, &exampleOutputDeleteDNSRecord)
}

//go:embed example_output_upsert_dns_record.json
var exampleOutputUpsertDNSRecordBytes []byte

var exampleOutputUpsertDNSRecordOnce sync.Once
var exampleOutputUpsertDNSRecord map[string]any

func (u *UpsertDNSRecord) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputUpsertDNSRecordOnce, exampleOutputUpsertDNSRecordBytes, &exampleOutputUpsertDNSRecord)
}

//go:embed example_output_create_load_balancer.json
var exampleOutputCreateLoadBalancerBytes []byte

var exampleOutputCreateLoadBalancerOnce sync.Once
var exampleOutputCreateLoadBalancer map[string]any

func (c *CreateLoadBalancer) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateLoadBalancerOnce, exampleOutputCreateLoadBalancerBytes, &exampleOutputCreateLoadBalancer)
}

//go:embed example_output_delete_load_balancer.json
var exampleOutputDeleteLoadBalancerBytes []byte

var exampleOutputDeleteLoadBalancerOnce sync.Once
var exampleOutputDeleteLoadBalancer map[string]any

func (d *DeleteLoadBalancer) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputDeleteLoadBalancerOnce, exampleOutputDeleteLoadBalancerBytes, &exampleOutputDeleteLoadBalancer)
}

//go:embed example_output_assign_reserved_ip.json
var exampleOutputAssignReservedIPBytes []byte

var exampleOutputAssignReservedIPOnce sync.Once
var exampleOutputAssignReservedIP map[string]any

func (a *AssignReservedIP) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputAssignReservedIPOnce, exampleOutputAssignReservedIPBytes, &exampleOutputAssignReservedIP)
}
