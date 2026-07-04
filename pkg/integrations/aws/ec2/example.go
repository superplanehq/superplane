package ec2

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_data_on_image.json
var exampleDataOnImageBytes []byte

//go:embed example_data_on_alarm.json
var exampleDataOnAlarmBytes []byte

//go:embed example_output_create_image.json
var exampleOutputCreateImageBytes []byte

//go:embed example_output_get_image.json
var exampleOutputGetImageBytes []byte

//go:embed example_output_copy_image.json
var exampleOutputCopyImageBytes []byte

//go:embed example_output_deregister_image.json
var exampleOutputDeregisterImageBytes []byte

//go:embed example_output_enable_image.json
var exampleOutputEnableImageBytes []byte

//go:embed example_output_disable_image.json
var exampleOutputDisableImageBytes []byte

//go:embed example_output_enable_image_deprecation.json
var exampleOutputEnableImageDeprecationBytes []byte

//go:embed example_output_disable_image_deprecation.json
var exampleOutputDisableImageDeprecationBytes []byte

//go:embed example_output_create_instance.json
var exampleOutputCreateInstanceBytes []byte

//go:embed example_output_delete_instance.json
var exampleOutputDeleteInstanceBytes []byte

//go:embed example_output_get_instance.json
var exampleOutputGetInstanceBytes []byte

//go:embed example_output_manage_instance_power.json
var exampleOutputManageInstancePowerBytes []byte

//go:embed example_output_get_instance_metrics.json
var exampleOutputGetInstanceMetricsBytes []byte

//go:embed example_output_update_instance.json
var exampleOutputUpdateInstanceBytes []byte

//go:embed example_output_create_alarm.json
var exampleOutputCreateAlarmBytes []byte

//go:embed example_output_get_alarm.json
var exampleOutputGetAlarmBytes []byte

//go:embed example_output_allocate_elastic_ip.json
var exampleOutputAllocateElasticIPBytes []byte

//go:embed example_output_release_elastic_ip.json
var exampleOutputReleaseElasticIPBytes []byte

//go:embed example_output_manage_elastic_ip.json
var exampleOutputManageElasticIPBytes []byte

//go:embed example_output_create_load_balancer.json
var exampleOutputCreateLoadBalancerBytes []byte

//go:embed example_output_delete_load_balancer.json
var exampleOutputDeleteLoadBalancerBytes []byte

//go:embed example_output_update_alarm.json
var exampleOutputUpdateAlarmBytes []byte

//go:embed example_output_delete_alarm.json
var exampleOutputDeleteAlarmBytes []byte

var exampleDataOnImageOnce sync.Once
var exampleDataOnImage map[string]any

var exampleDataOnAlarmOnce sync.Once
var exampleDataOnAlarm map[string]any

var exampleOutputCreateImageOnce sync.Once
var exampleOutputCreateImage map[string]any

var exampleOutputGetImageOnce sync.Once
var exampleOutputGetImage map[string]any

var exampleOutputCopyImageOnce sync.Once
var exampleOutputCopyImage map[string]any

var exampleOutputDeregisterImageOnce sync.Once
var exampleOutputDeregisterImage map[string]any

var exampleOutputEnableImageOnce sync.Once
var exampleOutputEnableImage map[string]any

var exampleOutputDisableImageOnce sync.Once
var exampleOutputDisableImage map[string]any

var exampleOutputEnableImageDeprecationOnce sync.Once
var exampleOutputEnableImageDeprecation map[string]any

var exampleOutputDisableImageDeprecationOnce sync.Once
var exampleOutputDisableImageDeprecation map[string]any

var exampleOutputCreateInstanceOnce sync.Once
var exampleOutputCreateInstance map[string]any

var exampleOutputDeleteInstanceOnce sync.Once
var exampleOutputDeleteInstance map[string]any

var exampleOutputGetInstanceOnce sync.Once
var exampleOutputGetInstance map[string]any

var exampleOutputManageInstancePowerOnce sync.Once
var exampleOutputManageInstancePower map[string]any

var exampleOutputGetInstanceMetricsOnce sync.Once
var exampleOutputGetInstanceMetrics map[string]any

var exampleOutputUpdateInstanceOnce sync.Once
var exampleOutputUpdateInstance map[string]any

var exampleOutputCreateAlarmOnce sync.Once
var exampleOutputCreateAlarm map[string]any

var exampleOutputGetAlarmOnce sync.Once
var exampleOutputGetAlarm map[string]any

var exampleOutputAllocateElasticIPOnce sync.Once
var exampleOutputAllocateElasticIP map[string]any

var exampleOutputReleaseElasticIPOnce sync.Once
var exampleOutputReleaseElasticIP map[string]any

var exampleOutputManageElasticIPOnce sync.Once
var exampleOutputManageElasticIP map[string]any

var exampleOutputCreateLoadBalancerOnce sync.Once
var exampleOutputCreateLoadBalancer map[string]any

var exampleOutputDeleteLoadBalancerOnce sync.Once
var exampleOutputDeleteLoadBalancer map[string]any

var exampleOutputUpdateAlarmOnce sync.Once
var exampleOutputUpdateAlarm map[string]any

var exampleOutputDeleteAlarmOnce sync.Once
var exampleOutputDeleteAlarm map[string]any

func (t *OnImage) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnImageOnce, exampleDataOnImageBytes, &exampleDataOnImage)
}

func (t *OnAlarm) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnAlarmOnce, exampleDataOnAlarmBytes, &exampleDataOnAlarm)
}

func (c *CreateImage) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateImageOnce, exampleOutputCreateImageBytes, &exampleOutputCreateImage)
}

func (c *GetImage) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetImageOnce, exampleOutputGetImageBytes, &exampleOutputGetImage)
}

func (c *CopyImage) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCopyImageOnce, exampleOutputCopyImageBytes, &exampleOutputCopyImage)
}

func (c *DeregisterImage) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputDeregisterImageOnce,
		exampleOutputDeregisterImageBytes,
		&exampleOutputDeregisterImage,
	)
}

func (c *EnableImage) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputEnableImageOnce, exampleOutputEnableImageBytes, &exampleOutputEnableImage)
}

func (c *DisableImage) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputDisableImageOnce, exampleOutputDisableImageBytes, &exampleOutputDisableImage)
}

func (c *EnableImageDeprecation) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputEnableImageDeprecationOnce,
		exampleOutputEnableImageDeprecationBytes,
		&exampleOutputEnableImageDeprecation,
	)
}

func (c *DisableImageDeprecation) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputDisableImageDeprecationOnce,
		exampleOutputDisableImageDeprecationBytes,
		&exampleOutputDisableImageDeprecation,
	)
}

func (c *CreateInstance) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputCreateInstanceOnce,
		exampleOutputCreateInstanceBytes,
		&exampleOutputCreateInstance,
	)
}

func (c *DeleteInstance) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputDeleteInstanceOnce,
		exampleOutputDeleteInstanceBytes,
		&exampleOutputDeleteInstance,
	)
}

func (c *GetInstance) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputGetInstanceOnce,
		exampleOutputGetInstanceBytes,
		&exampleOutputGetInstance,
	)
}

func (c *ManageInstancePower) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputManageInstancePowerOnce,
		exampleOutputManageInstancePowerBytes,
		&exampleOutputManageInstancePower,
	)
}

func (c *GetInstanceMetrics) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputGetInstanceMetricsOnce,
		exampleOutputGetInstanceMetricsBytes,
		&exampleOutputGetInstanceMetrics,
	)
}

func (c *UpdateInstance) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputUpdateInstanceOnce,
		exampleOutputUpdateInstanceBytes,
		&exampleOutputUpdateInstance,
	)
}

func (c *CreateAlarm) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputCreateAlarmOnce,
		exampleOutputCreateAlarmBytes,
		&exampleOutputCreateAlarm,
	)
}

func (c *GetAlarm) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputGetAlarmOnce,
		exampleOutputGetAlarmBytes,
		&exampleOutputGetAlarm,
	)
}

func (c *AllocateElasticIP) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputAllocateElasticIPOnce,
		exampleOutputAllocateElasticIPBytes,
		&exampleOutputAllocateElasticIP,
	)
}

func (c *ReleaseElasticIP) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputReleaseElasticIPOnce,
		exampleOutputReleaseElasticIPBytes,
		&exampleOutputReleaseElasticIP,
	)
}

func (c *ManageElasticIP) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputManageElasticIPOnce,
		exampleOutputManageElasticIPBytes,
		&exampleOutputManageElasticIP,
	)
}

func (c *CreateLoadBalancer) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputCreateLoadBalancerOnce,
		exampleOutputCreateLoadBalancerBytes,
		&exampleOutputCreateLoadBalancer,
	)
}

func (c *DeleteLoadBalancer) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputDeleteLoadBalancerOnce,
		exampleOutputDeleteLoadBalancerBytes,
		&exampleOutputDeleteLoadBalancer,
	)
}

func (c *UpdateAlarm) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputUpdateAlarmOnce,
		exampleOutputUpdateAlarmBytes,
		&exampleOutputUpdateAlarm,
	)
}

func (c *DeleteAlarm) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputDeleteAlarmOnce,
		exampleOutputDeleteAlarmBytes,
		&exampleOutputDeleteAlarm,
	)
}
