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
