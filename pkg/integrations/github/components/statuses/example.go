package statuses

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed payloads/publish_commit_status.json
var exampleOutputPublishCommitStatusBytes []byte

//go:embed payloads/on_commit_status.json
var exampleDataOnCommitStatusBytes []byte

var exampleOutputPublishCommitStatusOnce sync.Once
var exampleOutputPublishCommitStatus map[string]any

var exampleDataOnCommitStatusOnce sync.Once
var exampleDataOnCommitStatus map[string]any

func (c *PublishCommitStatus) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputPublishCommitStatusOnce,
		exampleOutputPublishCommitStatusBytes,
		&exampleOutputPublishCommitStatus,
	)
}

func (t *OnCommitStatus) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleDataOnCommitStatusOnce,
		exampleDataOnCommitStatusBytes,
		&exampleDataOnCommitStatus,
	)
}
