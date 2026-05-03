package contents

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed payloads/create_release.json
var exampleOutputCreateReleaseBytes []byte

//go:embed payloads/get_release.json
var exampleOutputGetReleaseBytes []byte

//go:embed payloads/update_release.json
var exampleOutputUpdateReleaseBytes []byte

//go:embed payloads/delete_release.json
var exampleOutputDeleteReleaseBytes []byte

//go:embed payloads/on_push.json
var exampleDataOnPushBytes []byte

//go:embed payloads/on_release.json
var exampleDataOnReleaseBytes []byte

//go:embed payloads/on_tag_created.json
var exampleDataOnTagCreatedBytes []byte

//go:embed payloads/on_branch_created.json
var exampleDataOnBranchCreatedBytes []byte

var exampleOutputCreateReleaseOnce sync.Once
var exampleOutputCreateRelease map[string]any

var exampleOutputGetReleaseOnce sync.Once
var exampleOutputGetRelease map[string]any

var exampleOutputUpdateReleaseOnce sync.Once
var exampleOutputUpdateRelease map[string]any

var exampleOutputDeleteReleaseOnce sync.Once
var exampleOutputDeleteRelease map[string]any

var exampleDataOnPushOnce sync.Once
var exampleDataOnPush map[string]any

var exampleDataOnReleaseOnce sync.Once
var exampleDataOnRelease map[string]any

var exampleDataOnTagCreatedOnce sync.Once
var exampleDataOnTagCreated map[string]any

var exampleDataOnBranchCreatedOnce sync.Once
var exampleDataOnBranchCreated map[string]any

func (c *CreateRelease) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateReleaseOnce, exampleOutputCreateReleaseBytes, &exampleOutputCreateRelease)
}

func (c *GetRelease) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetReleaseOnce, exampleOutputGetReleaseBytes, &exampleOutputGetRelease)
}

func (c *UpdateRelease) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputUpdateReleaseOnce, exampleOutputUpdateReleaseBytes, &exampleOutputUpdateRelease)
}

func (c *DeleteRelease) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputDeleteReleaseOnce, exampleOutputDeleteReleaseBytes, &exampleOutputDeleteRelease)
}

func (t *OnBranchCreated) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnBranchCreatedOnce, exampleDataOnBranchCreatedBytes, &exampleDataOnBranchCreated)
}

func (t *OnPush) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnPushOnce, exampleDataOnPushBytes, &exampleDataOnPush)
}

func (t *OnRelease) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnReleaseOnce, exampleDataOnReleaseBytes, &exampleDataOnRelease)
}

func (t *OnTagCreated) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnTagCreatedOnce, exampleDataOnTagCreatedBytes, &exampleDataOnTagCreated)
}
