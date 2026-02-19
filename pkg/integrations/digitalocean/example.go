package digitalocean

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_data_on_droplet_event.json
var exampleDataOnDropletEventBytes []byte

var exampleDataOnDropletEventOnce sync.Once
var exampleDataOnDropletEvent map[string]any

//go:embed example_output_create_droplet.json
var exampleOutputCreateDropletBytes []byte

var exampleOutputCreateDropletOnce sync.Once
var exampleOutputCreateDroplet map[string]any

func (t *OnDropletEvent) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnDropletEventOnce, exampleDataOnDropletEventBytes, &exampleDataOnDropletEvent)
}

func (c *CreateDroplet) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateDropletOnce, exampleOutputCreateDropletBytes, &exampleOutputCreateDroplet)
}
