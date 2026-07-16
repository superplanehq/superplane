package contents

import (
	_ "embed"

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
var exampleOutputCreateRelease = utils.NewEmbeddedJSON(exampleOutputCreateReleaseBytes)
var exampleOutputGetRelease = utils.NewEmbeddedJSON(exampleOutputGetReleaseBytes)
var exampleOutputUpdateRelease = utils.NewEmbeddedJSON(exampleOutputUpdateReleaseBytes)
var exampleOutputDeleteRelease = utils.NewEmbeddedJSON(exampleOutputDeleteReleaseBytes)
var exampleDataOnPush = utils.NewEmbeddedJSON(exampleDataOnPushBytes)
var exampleDataOnRelease = utils.NewEmbeddedJSON(exampleDataOnReleaseBytes)
var exampleDataOnTagCreated = utils.NewEmbeddedJSON(exampleDataOnTagCreatedBytes)
var exampleDataOnBranchCreated = utils.NewEmbeddedJSON(exampleDataOnBranchCreatedBytes)

func (c *CreateRelease) ExampleOutput() map[string]any {
	return exampleOutputCreateRelease.Value()
}

func (c *GetRelease) ExampleOutput() map[string]any {
	return exampleOutputGetRelease.Value()
}

func (c *UpdateRelease) ExampleOutput() map[string]any {
	return exampleOutputUpdateRelease.Value()
}

func (c *DeleteRelease) ExampleOutput() map[string]any {
	return exampleOutputDeleteRelease.Value()
}

func (t *OnBranchCreated) ExampleData() map[string]any {
	return exampleDataOnBranchCreated.Value()
}

func (t *OnPush) ExampleData() map[string]any {
	return exampleDataOnPush.Value()
}

func (t *OnRelease) ExampleData() map[string]any {
	return exampleDataOnRelease.Value()
}

func (t *OnTagCreated) ExampleData() map[string]any {
	return exampleDataOnTagCreated.Value()
}
