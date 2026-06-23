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

//go:embed example_output_get_package.json
var exampleOutputGetPackageBytes []byte

var exampleOutputGetPackageOnce sync.Once
var exampleOutputGetPackage map[string]any

func (g *GetPackage) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetPackageOnce, exampleOutputGetPackageBytes, &exampleOutputGetPackage)
}

//go:embed example_output_list_packages.json
var exampleOutputListPackagesBytes []byte

var exampleOutputListPackagesOnce sync.Once
var exampleOutputListPackages map[string]any

func (l *ListPackages) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputListPackagesOnce, exampleOutputListPackagesBytes, &exampleOutputListPackages)
}

//go:embed example_output_promote_package.json
var exampleOutputPromotePackageBytes []byte

var exampleOutputPromotePackageOnce sync.Once
var exampleOutputPromotePackage map[string]any

func (p *PromotePackage) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputPromotePackageOnce, exampleOutputPromotePackageBytes, &exampleOutputPromotePackage)
}

//go:embed example_data_on_security_scan_completed.json
var exampleDataOnSecurityScanCompletedBytes []byte

var exampleDataOnSecurityScanCompletedOnce sync.Once
var exampleDataOnSecurityScanCompleted map[string]any

func onSecurityScanCompletedExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnSecurityScanCompletedOnce, exampleDataOnSecurityScanCompletedBytes, &exampleDataOnSecurityScanCompleted)
}

//go:embed example_data_on_package_created.json
var exampleDataOnPackageCreatedBytes []byte

var exampleDataOnPackageCreatedOnce sync.Once
var exampleDataOnPackageCreated map[string]any

func onPackageCreatedExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnPackageCreatedOnce, exampleDataOnPackageCreatedBytes, &exampleDataOnPackageCreated)
}
