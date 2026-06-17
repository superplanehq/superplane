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

//go:embed example_output_get_package_compliance.json
var exampleOutputGetPackageComplianceBytes []byte

var exampleOutputGetPackageComplianceOnce sync.Once
var exampleOutputGetPackageCompliance map[string]any

func (g *GetPackageCompliance) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetPackageComplianceOnce, exampleOutputGetPackageComplianceBytes, &exampleOutputGetPackageCompliance)
}

//go:embed example_data_on_compliance_check_completed.json
var exampleDataOnComplianceCheckCompletedBytes []byte

var exampleDataOnComplianceCheckCompletedOnce sync.Once
var exampleDataOnComplianceCheckCompleted map[string]any

func onComplianceCheckCompletedExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnComplianceCheckCompletedOnce, exampleDataOnComplianceCheckCompletedBytes, &exampleDataOnComplianceCheckCompleted)
}
