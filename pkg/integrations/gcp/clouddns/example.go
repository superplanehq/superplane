package clouddns

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_create_record.json
var exampleOutputCreateRecordBytes []byte

//go:embed example_output_delete_record.json
var exampleOutputDeleteRecordBytes []byte

//go:embed example_output_update_record.json
var exampleOutputUpdateRecordBytes []byte

var (
	exampleOutputCreateRecordOnce sync.Once
	exampleOutputCreateRecord     map[string]any

	exampleOutputDeleteRecordOnce sync.Once
	exampleOutputDeleteRecord     map[string]any

	exampleOutputUpdateRecordOnce sync.Once
	exampleOutputUpdateRecord     map[string]any
)

func (c *CreateRecord) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateRecordOnce, exampleOutputCreateRecordBytes, &exampleOutputCreateRecord)
}

func (c *DeleteRecord) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputDeleteRecordOnce, exampleOutputDeleteRecordBytes, &exampleOutputDeleteRecord)
}

func (c *UpdateRecord) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputUpdateRecordOnce, exampleOutputUpdateRecordBytes, &exampleOutputUpdateRecord)
}
