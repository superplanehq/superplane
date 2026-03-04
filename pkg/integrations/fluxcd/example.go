package fluxcd

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_reconcile_source.json
var exampleOutputReconcileSourceBytes []byte

//go:embed example_data_on_reconciliation_completed.json
var exampleDataOnReconciliationCompletedBytes []byte

var exampleOutputReconcileSourceOnce sync.Once
var exampleOutputReconcileSource map[string]any

var exampleDataOnReconciliationCompletedOnce sync.Once
var exampleDataOnReconciliationCompleted map[string]any

func (c *ReconcileSource) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputReconcileSourceOnce, exampleOutputReconcileSourceBytes, &exampleOutputReconcileSource)
}

func (t *OnReconciliationCompleted) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnReconciliationCompletedOnce, exampleDataOnReconciliationCompletedBytes, &exampleDataOnReconciliationCompleted)
}
