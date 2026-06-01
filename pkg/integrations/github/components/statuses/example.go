package statuses

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed payloads/publish_commit_status.json
var exampleOutputPublishCommitStatusBytes []byte

//go:embed payloads/on_status.json
var exampleDataOnStatusBytes []byte

var exampleOutputPublishCommitStatusOnce sync.Once
var exampleOutputPublishCommitStatus map[string]any

var exampleDataOnStatusOnce sync.Once
var exampleDataOnStatus map[string]any

func (c *PublishCommitStatus) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputPublishCommitStatusOnce,
		exampleOutputPublishCommitStatusBytes,
		&exampleOutputPublishCommitStatus,
	)
}

func (t *OnStatus) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleDataOnStatusOnce,
		exampleDataOnStatusBytes,
		&exampleDataOnStatus,
	)
}
