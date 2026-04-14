package hashicorp_vault

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_get_secret.json
var exampleOutputGetSecretBytes []byte

var exampleOutputGetSecretOnce sync.Once
var exampleOutputGetSecret map[string]any

func (c *getSecret) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetSecretOnce, exampleOutputGetSecretBytes, &exampleOutputGetSecret)
}
