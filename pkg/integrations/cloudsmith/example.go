package cloudsmith

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_get_repository.json
var exampleOutputGetRepositoryBytes []byte

//go:embed example_output_get_package.json
var exampleOutputGetPackageBytes []byte

//go:embed example_output_resync_package.json
var exampleOutputResyncPackageBytes []byte

//go:embed example_output_tag_package.json
var exampleOutputTagPackageBytes []byte

//go:embed example_output_delete_package.json
var exampleOutputDeletePackageBytes []byte

//go:embed example_data_on_security_scan_completed.json
var exampleDataOnSecurityScanCompletedBytes []byte

//go:embed example_data_on_package_created.json
var exampleDataOnPackageCreatedBytes []byte

var exampleOutputGetRepositoryOnce sync.Once
var exampleOutputGetRepository map[string]any
var exampleOutputGetPackageOnce sync.Once
var exampleOutputGetPackage map[string]any
var exampleOutputResyncPackageOnce sync.Once
var exampleOutputResyncPackage map[string]any
var exampleOutputTagPackageOnce sync.Once
var exampleOutputTagPackage map[string]any
var exampleOutputDeletePackageOnce sync.Once
var exampleOutputDeletePackage map[string]any
var exampleDataOnSecurityScanCompletedOnce sync.Once
var exampleDataOnSecurityScanCompleted map[string]any
var exampleDataOnPackageCreatedOnce sync.Once
var exampleDataOnPackageCreated map[string]any

func (g *GetRepository) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetRepositoryOnce, exampleOutputGetRepositoryBytes, &exampleOutputGetRepository)
}

func (g *GetPackage) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetPackageOnce, exampleOutputGetPackageBytes, &exampleOutputGetPackage)
}

func (r *ResyncPackage) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputResyncPackageOnce, exampleOutputResyncPackageBytes, &exampleOutputResyncPackage)
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

func (t *TagPackage) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputTagPackageOnce, exampleOutputTagPackageBytes, &exampleOutputTagPackage)
}

func (d *DeletePackage) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputDeletePackageOnce, exampleOutputDeletePackageBytes, &exampleOutputDeletePackage)
}

func onSecurityScanCompletedExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnSecurityScanCompletedOnce, exampleDataOnSecurityScanCompletedBytes, &exampleDataOnSecurityScanCompleted)
}

func onPackageCreatedExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnPackageCreatedOnce, exampleDataOnPackageCreatedBytes, &exampleDataOnPackageCreated)
}
