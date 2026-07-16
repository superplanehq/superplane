package statuses

import (
	_ "embed"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed payloads/publish_commit_status.json
var exampleOutputPublishCommitStatusBytes []byte

//go:embed payloads/get_combined_commit_status.json
var exampleOutputGetCombinedCommitStatusBytes []byte

//go:embed payloads/on_commit_status.json
var exampleDataOnCommitStatusBytes []byte
var exampleOutputPublishCommitStatus = utils.NewEmbeddedJSON(exampleOutputPublishCommitStatusBytes)
var exampleOutputGetCombinedCommitStatus = utils.NewEmbeddedJSON(exampleOutputGetCombinedCommitStatusBytes)
var exampleDataOnCommitStatus = utils.NewEmbeddedJSON(exampleDataOnCommitStatusBytes)

func (c *PublishCommitStatus) ExampleOutput() map[string]any {
	return exampleOutputPublishCommitStatus.Value()
}

func (c *GetCombinedCommitStatus) ExampleOutput() map[string]any {
	return exampleOutputGetCombinedCommitStatus.Value()
}

func (t *OnCommitStatus) ExampleData() map[string]any {
	return exampleDataOnCommitStatus.Value()
}
