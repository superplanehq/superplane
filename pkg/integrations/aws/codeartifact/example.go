package codeartifact

import (
	_ "embed"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_data_on_package_version.json
var exampleDataOnPackageVersionBytes []byte

//go:embed example_data_get_package_version.json
var exampleOutputGetPackageVersionBytes []byte

//go:embed example_output_create_repository.json
var exampleOutputCreateRepositoryBytes []byte

//go:embed example_output_delete_repository.json
var exampleOutputDeleteRepositoryBytes []byte

//go:embed example_output_update_package_versions_status.json
var exampleOutputUpdatePackageVersionsStatusBytes []byte

//go:embed example_output_copy_package_versions.json
var exampleOutputCopyPackageVersionsBytes []byte

//go:embed example_output_delete_package_versions.json
var exampleOutputDeletePackageVersionsBytes []byte

//go:embed example_output_dispose_package_versions.json
var exampleOutputDisposePackageVersionsBytes []byte
var exampleDataOnPackageVersion = utils.NewEmbeddedJSON(exampleDataOnPackageVersionBytes)
var exampleOutputGetPackageVersion = utils.NewEmbeddedJSON(exampleOutputGetPackageVersionBytes)
var exampleOutputCreateRepository = utils.NewEmbeddedJSON(exampleOutputCreateRepositoryBytes)
var exampleOutputDeleteRepository = utils.NewEmbeddedJSON(exampleOutputDeleteRepositoryBytes)
var exampleOutputUpdatePackageVersionsStatus = utils.NewEmbeddedJSON(exampleOutputUpdatePackageVersionsStatusBytes)
var exampleOutputCopyPackageVersions = utils.NewEmbeddedJSON(exampleOutputCopyPackageVersionsBytes)
var exampleOutputDeletePackageVersions = utils.NewEmbeddedJSON(exampleOutputDeletePackageVersionsBytes)
var exampleOutputDisposePackageVersions = utils.NewEmbeddedJSON(exampleOutputDisposePackageVersionsBytes)

func (t *OnPackageVersion) ExampleData() map[string]any {
	return exampleDataOnPackageVersion.Value()
}

func (c *GetPackageVersion) ExampleOutput() map[string]any {
	return exampleOutputGetPackageVersion.Value()
}

func (c *CreateRepository) ExampleOutput() map[string]any {
	return exampleOutputCreateRepository.Value()
}

func (c *DeleteRepository) ExampleOutput() map[string]any {
	return exampleOutputDeleteRepository.Value()
}

func (c *UpdatePackageVersionsStatus) ExampleOutput() map[string]any {
	return exampleOutputUpdatePackageVersionsStatus.Value()
}

func (c *CopyPackageVersions) ExampleOutput() map[string]any {
	return exampleOutputCopyPackageVersions.Value()
}

func (c *DeletePackageVersions) ExampleOutput() map[string]any {
	return exampleOutputDeletePackageVersions.Value()
}

func (c *DisposePackageVersions) ExampleOutput() map[string]any {
	return exampleOutputDisposePackageVersions.Value()
}
