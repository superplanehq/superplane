package metadata

import (
	_ "embed"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed payloads/get_repository_permission.json
var exampleOutputGetRepositoryPermissionBytes []byte
var exampleOutputGetRepositoryPermission = utils.NewEmbeddedJSON(exampleOutputGetRepositoryPermissionBytes)

func (c *GetRepositoryPermission) ExampleOutput() map[string]any {
	return exampleOutputGetRepositoryPermission.Value()
}
