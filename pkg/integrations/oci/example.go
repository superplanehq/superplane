package oci

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_create_compute_instance.json
var exampleOutputCreateComputeInstanceBytes []byte

//go:embed example_output_create_image.json
var exampleOutputCreateImageBytes []byte

//go:embed example_output_get_image.json
var exampleOutputGetImageBytes []byte

//go:embed example_output_update_image.json
var exampleOutputUpdateImageBytes []byte

//go:embed example_output_delete_image.json
var exampleOutputDeleteImageBytes []byte

var exampleOutputCreateComputeInstanceOnce sync.Once
var exampleOutputCreateComputeInstanceCache map[string]any
var exampleOutputCreateImageOnce sync.Once
var exampleOutputCreateImageCache map[string]any
var exampleOutputGetImageOnce sync.Once
var exampleOutputGetImageCache map[string]any
var exampleOutputUpdateImageOnce sync.Once
var exampleOutputUpdateImageCache map[string]any
var exampleOutputDeleteImageOnce sync.Once
var exampleOutputDeleteImageCache map[string]any

func exampleOutputCreateComputeInstance() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateComputeInstanceOnce, exampleOutputCreateComputeInstanceBytes, &exampleOutputCreateComputeInstanceCache)
}

func exampleOutputCreateImage() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateImageOnce, exampleOutputCreateImageBytes, &exampleOutputCreateImageCache)
}

func exampleOutputGetImage() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetImageOnce, exampleOutputGetImageBytes, &exampleOutputGetImageCache)
}

func exampleOutputUpdateImage() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputUpdateImageOnce, exampleOutputUpdateImageBytes, &exampleOutputUpdateImageCache)
}

func exampleOutputDeleteImage() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputDeleteImageOnce, exampleOutputDeleteImageBytes, &exampleOutputDeleteImageCache)
}

//go:embed example_data_on_compute_instance_created.json
var exampleDataOnComputeInstanceCreatedBytes []byte

var exampleDataOnComputeInstanceCreatedOnce sync.Once
var exampleDataOnComputeInstanceCreatedCache map[string]any

func exampleDataOnComputeInstanceCreated() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnComputeInstanceCreatedOnce, exampleDataOnComputeInstanceCreatedBytes, &exampleDataOnComputeInstanceCreatedCache)
}
