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
