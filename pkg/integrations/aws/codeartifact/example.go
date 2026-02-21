package codeartifact

import (
	_ "embed"
	"sync"

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

var exampleDataOnPackageVersionOnce sync.Once
var exampleDataOnPackageVersion map[string]any

var exampleOutputGetPackageVersionOnce sync.Once
var exampleOutputGetPackageVersion map[string]any

var exampleOutputCreateRepositoryOnce sync.Once
var exampleOutputCreateRepository map[string]any

var exampleOutputDeleteRepositoryOnce sync.Once
var exampleOutputDeleteRepository map[string]any

var exampleOutputUpdatePackageVersionsStatusOnce sync.Once
var exampleOutputUpdatePackageVersionsStatus map[string]any

var exampleOutputCopyPackageVersionsOnce sync.Once
var exampleOutputCopyPackageVersions map[string]any

var exampleOutputDeletePackageVersionsOnce sync.Once
var exampleOutputDeletePackageVersions map[string]any

var exampleOutputDisposePackageVersionsOnce sync.Once
var exampleOutputDisposePackageVersions map[string]any

func (t *OnPackageVersion) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnPackageVersionOnce, exampleDataOnPackageVersionBytes, &exampleDataOnPackageVersion)
}

func (c *GetPackageVersion) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputGetPackageVersionOnce,
		exampleOutputGetPackageVersionBytes,
		&exampleOutputGetPackageVersion,
	)
}

func (c *CreateRepository) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputCreateRepositoryOnce,
		exampleOutputCreateRepositoryBytes,
		&exampleOutputCreateRepository,
	)
}

func (c *DeleteRepository) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputDeleteRepositoryOnce,
		exampleOutputDeleteRepositoryBytes,
		&exampleOutputDeleteRepository,
	)
}

func (c *UpdatePackageVersionsStatus) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputUpdatePackageVersionsStatusOnce,
		exampleOutputUpdatePackageVersionsStatusBytes,
		&exampleOutputUpdatePackageVersionsStatus,
	)
}

func (c *CopyPackageVersions) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputCopyPackageVersionsOnce,
		exampleOutputCopyPackageVersionsBytes,
		&exampleOutputCopyPackageVersions,
	)
}

func (c *DeletePackageVersions) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputDeletePackageVersionsOnce,
		exampleOutputDeletePackageVersionsBytes,
		&exampleOutputDeletePackageVersions,
	)
}

func (c *DisposePackageVersions) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputDisposePackageVersionsOnce,
		exampleOutputDisposePackageVersionsBytes,
		&exampleOutputDisposePackageVersions,
	)
}
