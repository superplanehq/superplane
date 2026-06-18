package cloudsmith

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_get_repository.json
var exampleOutputGetRepositoryBytes []byte

var exampleOutputGetRepositoryOnce sync.Once
var exampleOutputGetRepository map[string]any

func (g *GetRepository) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetRepositoryOnce, exampleOutputGetRepositoryBytes, &exampleOutputGetRepository)
}

//go:embed example_data_on_compliance_check_completed.json
var exampleDataOnComplianceCheckCompletedBytes []byte

var exampleDataOnComplianceCheckCompletedOnce sync.Once
var exampleDataOnComplianceCheckCompleted map[string]any

func onComplianceCheckCompletedExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnComplianceCheckCompletedOnce, exampleDataOnComplianceCheckCompletedBytes, &exampleDataOnComplianceCheckCompleted)
}

//go:embed example_data_on_package_created.json
var exampleDataOnPackageCreatedBytes []byte

var exampleDataOnPackageCreatedOnce sync.Once
var exampleDataOnPackageCreated map[string]any

func onPackageCreatedExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnPackageCreatedOnce, exampleDataOnPackageCreatedBytes, &exampleDataOnPackageCreated)
}
