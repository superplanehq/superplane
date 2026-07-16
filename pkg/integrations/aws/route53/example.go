package route53

import (
	_ "embed"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_create_record.json
var exampleOutputCreateRecordBytes []byte

//go:embed example_output_upsert_record.json
var exampleOutputUpsertRecordBytes []byte

//go:embed example_output_delete_record.json
var exampleOutputDeleteRecordBytes []byte
var exampleOutputCreateRecord = utils.NewEmbeddedJSON(exampleOutputCreateRecordBytes)
var exampleOutputUpsertRecord = utils.NewEmbeddedJSON(exampleOutputUpsertRecordBytes)
var exampleOutputDeleteRecord = utils.NewEmbeddedJSON(exampleOutputDeleteRecordBytes)

func (c *CreateRecord) ExampleOutput() map[string]any {
	return exampleOutputCreateRecord.Value()
}

func (c *UpsertRecord) ExampleOutput() map[string]any {
	return exampleOutputUpsertRecord.Value()
}

func (c *DeleteRecord) ExampleOutput() map[string]any {
	return exampleOutputDeleteRecord.Value()
}
