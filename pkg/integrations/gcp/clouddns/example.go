package clouddns

import (
	_ "embed"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_create_record.json
var exampleOutputCreateRecordBytes []byte

//go:embed example_output_delete_record.json
var exampleOutputDeleteRecordBytes []byte

//go:embed example_output_update_record.json
var exampleOutputUpdateRecordBytes []byte

var (
	exampleOutputCreateRecord = utils.NewEmbeddedJSON(exampleOutputCreateRecordBytes)
	exampleOutputDeleteRecord = utils.NewEmbeddedJSON(exampleOutputDeleteRecordBytes)
	exampleOutputUpdateRecord = utils.NewEmbeddedJSON(exampleOutputUpdateRecordBytes)
)

func (c *CreateRecord) ExampleOutput() map[string]any {
	return exampleOutputCreateRecord.Value()
}

func (c *DeleteRecord) ExampleOutput() map[string]any {
	return exampleOutputDeleteRecord.Value()
}

func (c *UpdateRecord) ExampleOutput() map[string]any {
	return exampleOutputUpdateRecord.Value()
}
