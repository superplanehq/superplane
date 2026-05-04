package statuses

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed payloads/publish_commit_status.json
var exampleOutputPublishCommitStatusBytes []byte

var exampleOutputPublishCommitStatusOnce sync.Once
var exampleOutputPublishCommitStatus map[string]any

func (c *PublishCommitStatus) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputPublishCommitStatusOnce,
		exampleOutputPublishCommitStatusBytes,
		&exampleOutputPublishCommitStatus,
	)
}
