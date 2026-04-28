package telemetry

import (
	"context"

	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/gorm"
)

const (
	createMetricsCallbackName = "telemetry:record_created_rows"
	updateMetricsCallbackName = "telemetry:record_updated_rows"
	deleteMetricsCallbackName = "telemetry:record_deleted_rows"

	dbOperationCreate = "create"
	dbOperationUpdate = "update"
	dbOperationDelete = "delete"
)

func registerDBOperationMetricsCallbacks() error {
	db := database.Conn()
	var err error

	if db.Callback().Create().Get(createMetricsCallbackName) == nil {
		err = db.Callback().Create().After("gorm:create").Register(createMetricsCallbackName, recordCreatedRowsMetric)
		if err != nil {
			return err
		}
	}

	if db.Callback().Update().Get(updateMetricsCallbackName) == nil {
		err = db.Callback().Update().After("gorm:update").Register(updateMetricsCallbackName, recordUpdatedRowsMetric)
		if err != nil {
			return err
		}
	}

	if db.Callback().Delete().Get(deleteMetricsCallbackName) == nil {
		err = db.Callback().Delete().After("gorm:delete").Register(deleteMetricsCallbackName, recordDeletedRowsMetric)
		if err != nil {
			return err
		}
	}

	return nil
}

func recordCreatedRowsMetric(tx *gorm.DB) {
	recordRowsAffectedMetric(tx, dbOperationCreate)
}

func recordUpdatedRowsMetric(tx *gorm.DB) {
	recordRowsAffectedMetric(tx, dbOperationUpdate)
}

func recordDeletedRowsMetric(tx *gorm.DB) {
	recordRowsAffectedMetric(tx, dbOperationDelete)
}

func recordRowsAffectedMetric(tx *gorm.DB, operation string) {
	if tx == nil || tx.Statement == nil || tx.RowsAffected <= 0 {
		return
	}

	tableName := tx.Statement.Table
	if tableName == "" && tx.Statement.Schema != nil {
		tableName = tx.Statement.Schema.Table
	}
	if tableName == "" {
		return
	}

	ctx := tx.Statement.Context
	if ctx == nil {
		ctx = context.Background()
	}

	RecordDBRowsAffected(ctx, tx.RowsAffected, tableName, operation)
}
