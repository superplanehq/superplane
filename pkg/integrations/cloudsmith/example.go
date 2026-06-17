package cloudsmith

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_get_repository.json
var exampleOutputGetRepositoryBytes []byte

//go:embed example_output_get_package_status.json
var exampleOutputGetPackageStatusBytes []byte

//go:embed example_output_get_package.json
var exampleOutputGetPackageBytes []byte

var exampleOutputGetRepositoryOnce sync.Once
var exampleOutputGetRepository map[string]any

var exampleOutputGetPackageStatusOnce sync.Once
var exampleOutputGetPackageStatus map[string]any

var exampleOutputGetPackageOnce sync.Once
var exampleOutputGetPackage map[string]any

func (g *GetRepository) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetRepositoryOnce, exampleOutputGetRepositoryBytes, &exampleOutputGetRepository)
}

func (g *GetPackageStatus) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetPackageStatusOnce, exampleOutputGetPackageStatusBytes, &exampleOutputGetPackageStatus)
}

func (g *GetPackage) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetPackageOnce, exampleOutputGetPackageBytes, &exampleOutputGetPackage)
}
