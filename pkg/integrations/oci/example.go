package oci

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_create_compute_instance.json
var exampleOutputCreateComputeInstanceBytes []byte

var exampleOutputCreateComputeInstanceOnce sync.Once
var exampleOutputCreateComputeInstanceCache map[string]any

func exampleOutputCreateComputeInstance() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateComputeInstanceOnce, exampleOutputCreateComputeInstanceBytes, &exampleOutputCreateComputeInstanceCache)
}

//go:embed example_data_on_compute_instance_created.json
var exampleDataOnComputeInstanceCreatedBytes []byte

var exampleDataOnComputeInstanceCreatedOnce sync.Once
var exampleDataOnComputeInstanceCreatedCache map[string]any

func exampleDataOnComputeInstanceCreated() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnComputeInstanceCreatedOnce, exampleDataOnComputeInstanceCreatedBytes, &exampleDataOnComputeInstanceCreatedCache)
}
