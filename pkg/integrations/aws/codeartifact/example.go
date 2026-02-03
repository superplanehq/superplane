package codeartifact

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_data_on_package_version.json
var exampleDataOnPackageVersionBytes []byte

var exampleDataOnPackageVersionOnce sync.Once
var exampleDataOnPackageVersion map[string]any

func (t *OnPackageVersion) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnPackageVersionOnce, exampleDataOnPackageVersionBytes, &exampleDataOnPackageVersion)
}
