package route53

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_create_record.json
var exampleOutputCreateRecordBytes []byte

//go:embed example_output_upsert_record.json
var exampleOutputUpsertRecordBytes []byte

//go:embed example_output_delete_record.json
var exampleOutputDeleteRecordBytes []byte

var exampleOutputCreateRecordOnce sync.Once
var exampleOutputCreateRecord map[string]any

var exampleOutputUpsertRecordOnce sync.Once
var exampleOutputUpsertRecord map[string]any

var exampleOutputDeleteRecordOnce sync.Once
var exampleOutputDeleteRecord map[string]any

func (c *CreateRecord) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputCreateRecordOnce,
		exampleOutputCreateRecordBytes,
		&exampleOutputCreateRecord,
	)
}

func (c *UpsertRecord) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputUpsertRecordOnce,
		exampleOutputUpsertRecordBytes,
		&exampleOutputUpsertRecord,
	)
}

func (c *DeleteRecord) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputDeleteRecordOnce,
		exampleOutputDeleteRecordBytes,
		&exampleOutputDeleteRecord,
	)
}
