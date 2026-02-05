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

var exampleDataOnPackageVersionOnce sync.Once
var exampleDataOnPackageVersion map[string]any

var exampleOutputGetPackageVersionOnce sync.Once
var exampleOutputGetPackageVersion map[string]any

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
