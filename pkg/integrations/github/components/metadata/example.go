package metadata

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed payloads/get_repository_permission.json
var exampleOutputGetRepositoryPermissionBytes []byte

var exampleOutputGetRepositoryPermissionOnce sync.Once
var exampleOutputGetRepositoryPermission map[string]any

func (c *GetRepositoryPermission) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputGetRepositoryPermissionOnce,
		exampleOutputGetRepositoryPermissionBytes,
		&exampleOutputGetRepositoryPermission,
	)
}
