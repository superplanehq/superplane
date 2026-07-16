package cloudsmith

import (
	_ "embed"

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

//go:embed example_output_scan_package.json
var exampleOutputScanPackageBytes []byte

//go:embed example_output_quarantine_package.json
var exampleOutputQuarantinePackageBytes []byte

//go:embed example_output_get_package_vulnerabilities.json
var exampleOutputGetPackageVulnerabilitiesBytes []byte

//go:embed example_data_on_security_scan_completed.json
var exampleDataOnSecurityScanCompletedBytes []byte

//go:embed example_data_on_package_created.json
var exampleDataOnPackageCreatedBytes []byte
var exampleOutputGetRepository = utils.NewEmbeddedJSON(exampleOutputGetRepositoryBytes)
var exampleOutputGetPackage = utils.NewEmbeddedJSON(exampleOutputGetPackageBytes)
var exampleOutputResyncPackage = utils.NewEmbeddedJSON(exampleOutputResyncPackageBytes)
var exampleOutputTagPackage = utils.NewEmbeddedJSON(exampleOutputTagPackageBytes)
var exampleOutputDeletePackage = utils.NewEmbeddedJSON(exampleOutputDeletePackageBytes)
var exampleOutputScanPackage = utils.NewEmbeddedJSON(exampleOutputScanPackageBytes)
var exampleOutputQuarantinePackage = utils.NewEmbeddedJSON(exampleOutputQuarantinePackageBytes)
var exampleOutputGetPackageVulnerabilities = utils.NewEmbeddedJSON(exampleOutputGetPackageVulnerabilitiesBytes)
var exampleDataOnSecurityScanCompleted = utils.NewEmbeddedJSON(exampleDataOnSecurityScanCompletedBytes)
var exampleDataOnPackageCreated = utils.NewEmbeddedJSON(exampleDataOnPackageCreatedBytes)

func (g *GetRepository) ExampleOutput() map[string]any {
	return exampleOutputGetRepository.Value()
}

func (g *GetPackage) ExampleOutput() map[string]any {
	return exampleOutputGetPackage.Value()
}

func (r *ResyncPackage) ExampleOutput() map[string]any {
	return exampleOutputResyncPackage.Value()
}

//go:embed example_output_list_packages.json
var exampleOutputListPackagesBytes []byte
var exampleOutputListPackages = utils.NewEmbeddedJSON(exampleOutputListPackagesBytes)

func (l *ListPackages) ExampleOutput() map[string]any {
	return exampleOutputListPackages.Value()
}

//go:embed example_output_promote_package.json
var exampleOutputPromotePackageBytes []byte
var exampleOutputPromotePackage = utils.NewEmbeddedJSON(exampleOutputPromotePackageBytes)

func (p *PromotePackage) ExampleOutput() map[string]any {
	return exampleOutputPromotePackage.Value()
}

func (t *TagPackage) ExampleOutput() map[string]any {
	return exampleOutputTagPackage.Value()
}

func (d *DeletePackage) ExampleOutput() map[string]any {
	return exampleOutputDeletePackage.Value()
}

func (s *ScanPackage) ExampleOutput() map[string]any {
	return exampleOutputScanPackage.Value()
}

func (q *QuarantinePackage) ExampleOutput() map[string]any {
	return exampleOutputQuarantinePackage.Value()
}

func (g *GetPackageVulnerabilities) ExampleOutput() map[string]any {
	return exampleOutputGetPackageVulnerabilities.Value()
}

func onSecurityScanCompletedExampleData() map[string]any {
	return exampleDataOnSecurityScanCompleted.Value()
}

func onPackageCreatedExampleData() map[string]any {
	return exampleDataOnPackageCreated.Value()
}

//go:embed example_output_create_vulnerability_policy.json
var exampleOutputCreateVulnerabilityPolicyBytes []byte
var exampleOutputCreateVulnerabilityPolicy = utils.NewEmbeddedJSON(exampleOutputCreateVulnerabilityPolicyBytes)

func (c *CreateVulnerabilityPolicy) ExampleOutput() map[string]any {
	return exampleOutputCreateVulnerabilityPolicy.Value()
}

//go:embed example_output_get_vulnerability_policy.json
var exampleOutputGetVulnerabilityPolicyBytes []byte
var exampleOutputGetVulnerabilityPolicy = utils.NewEmbeddedJSON(exampleOutputGetVulnerabilityPolicyBytes)

func (g *GetVulnerabilityPolicy) ExampleOutput() map[string]any {
	return exampleOutputGetVulnerabilityPolicy.Value()
}

//go:embed example_output_delete_vulnerability_policy.json
var exampleOutputDeleteVulnerabilityPolicyBytes []byte
var exampleOutputDeleteVulnerabilityPolicy = utils.NewEmbeddedJSON(exampleOutputDeleteVulnerabilityPolicyBytes)

func (d *DeleteVulnerabilityPolicy) ExampleOutput() map[string]any {
	return exampleOutputDeleteVulnerabilityPolicy.Value()
}
