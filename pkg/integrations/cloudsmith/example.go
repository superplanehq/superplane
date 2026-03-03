package cloudsmith

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed examples/package_synced.json
var exampleDataOnPackageEventBytes []byte

//go:embed examples/get_package_output.json
var exampleOutputGetPackageBytes []byte

var exampleDataOnPackageEventOnce sync.Once
var exampleDataOnPackageEvent map[string]any

var exampleOutputGetPackageOnce sync.Once
var exampleOutputGetPackage map[string]any

func onPackageEventExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnPackageEventOnce, exampleDataOnPackageEventBytes, &exampleDataOnPackageEvent)
}

func getPackageExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetPackageOnce, exampleOutputGetPackageBytes, &exampleOutputGetPackage)
}
