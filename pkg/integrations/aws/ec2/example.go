package ec2

import (
	_ "embed"

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
var exampleDataOnImage = utils.NewEmbeddedJSON(exampleDataOnImageBytes)
var exampleDataOnAlarm = utils.NewEmbeddedJSON(exampleDataOnAlarmBytes)
var exampleOutputCreateImage = utils.NewEmbeddedJSON(exampleOutputCreateImageBytes)
var exampleOutputGetImage = utils.NewEmbeddedJSON(exampleOutputGetImageBytes)
var exampleOutputCopyImage = utils.NewEmbeddedJSON(exampleOutputCopyImageBytes)
var exampleOutputDeregisterImage = utils.NewEmbeddedJSON(exampleOutputDeregisterImageBytes)
var exampleOutputEnableImage = utils.NewEmbeddedJSON(exampleOutputEnableImageBytes)
var exampleOutputDisableImage = utils.NewEmbeddedJSON(exampleOutputDisableImageBytes)
var exampleOutputEnableImageDeprecation = utils.NewEmbeddedJSON(exampleOutputEnableImageDeprecationBytes)
var exampleOutputDisableImageDeprecation = utils.NewEmbeddedJSON(exampleOutputDisableImageDeprecationBytes)
var exampleOutputCreateInstance = utils.NewEmbeddedJSON(exampleOutputCreateInstanceBytes)
var exampleOutputDeleteInstance = utils.NewEmbeddedJSON(exampleOutputDeleteInstanceBytes)
var exampleOutputGetInstance = utils.NewEmbeddedJSON(exampleOutputGetInstanceBytes)
var exampleOutputManageInstancePower = utils.NewEmbeddedJSON(exampleOutputManageInstancePowerBytes)
var exampleOutputGetInstanceMetrics = utils.NewEmbeddedJSON(exampleOutputGetInstanceMetricsBytes)
var exampleOutputUpdateInstance = utils.NewEmbeddedJSON(exampleOutputUpdateInstanceBytes)
var exampleOutputCreateAlarm = utils.NewEmbeddedJSON(exampleOutputCreateAlarmBytes)
var exampleOutputGetAlarm = utils.NewEmbeddedJSON(exampleOutputGetAlarmBytes)
var exampleOutputAllocateElasticIP = utils.NewEmbeddedJSON(exampleOutputAllocateElasticIPBytes)
var exampleOutputReleaseElasticIP = utils.NewEmbeddedJSON(exampleOutputReleaseElasticIPBytes)
var exampleOutputManageElasticIP = utils.NewEmbeddedJSON(exampleOutputManageElasticIPBytes)
var exampleOutputCreateLoadBalancer = utils.NewEmbeddedJSON(exampleOutputCreateLoadBalancerBytes)
var exampleOutputDeleteLoadBalancer = utils.NewEmbeddedJSON(exampleOutputDeleteLoadBalancerBytes)
var exampleOutputUpdateAlarm = utils.NewEmbeddedJSON(exampleOutputUpdateAlarmBytes)
var exampleOutputDeleteAlarm = utils.NewEmbeddedJSON(exampleOutputDeleteAlarmBytes)

func (t *OnImage) ExampleData() map[string]any {
	return exampleDataOnImage.Value()
}

func (t *OnAlarm) ExampleData() map[string]any {
	return exampleDataOnAlarm.Value()
}

func (c *CreateImage) ExampleOutput() map[string]any {
	return exampleOutputCreateImage.Value()
}

func (c *GetImage) ExampleOutput() map[string]any {
	return exampleOutputGetImage.Value()
}

func (c *CopyImage) ExampleOutput() map[string]any {
	return exampleOutputCopyImage.Value()
}

func (c *DeregisterImage) ExampleOutput() map[string]any {
	return exampleOutputDeregisterImage.Value()
}

func (c *EnableImage) ExampleOutput() map[string]any {
	return exampleOutputEnableImage.Value()
}

func (c *DisableImage) ExampleOutput() map[string]any {
	return exampleOutputDisableImage.Value()
}

func (c *EnableImageDeprecation) ExampleOutput() map[string]any {
	return exampleOutputEnableImageDeprecation.Value()
}

func (c *DisableImageDeprecation) ExampleOutput() map[string]any {
	return exampleOutputDisableImageDeprecation.Value()
}

func (c *CreateInstance) ExampleOutput() map[string]any {
	return exampleOutputCreateInstance.Value()
}

func (c *DeleteInstance) ExampleOutput() map[string]any {
	return exampleOutputDeleteInstance.Value()
}

func (c *GetInstance) ExampleOutput() map[string]any {
	return exampleOutputGetInstance.Value()
}

func (c *ManageInstancePower) ExampleOutput() map[string]any {
	return exampleOutputManageInstancePower.Value()
}

func (c *GetInstanceMetrics) ExampleOutput() map[string]any {
	return exampleOutputGetInstanceMetrics.Value()
}

func (c *UpdateInstance) ExampleOutput() map[string]any {
	return exampleOutputUpdateInstance.Value()
}

func (c *CreateAlarm) ExampleOutput() map[string]any {
	return exampleOutputCreateAlarm.Value()
}

func (c *GetAlarm) ExampleOutput() map[string]any {
	return exampleOutputGetAlarm.Value()
}

func (c *AllocateElasticIP) ExampleOutput() map[string]any {
	return exampleOutputAllocateElasticIP.Value()
}

func (c *ReleaseElasticIP) ExampleOutput() map[string]any {
	return exampleOutputReleaseElasticIP.Value()
}

func (c *ManageElasticIP) ExampleOutput() map[string]any {
	return exampleOutputManageElasticIP.Value()
}

func (c *CreateLoadBalancer) ExampleOutput() map[string]any {
	return exampleOutputCreateLoadBalancer.Value()
}

func (c *DeleteLoadBalancer) ExampleOutput() map[string]any {
	return exampleOutputDeleteLoadBalancer.Value()
}

func (c *UpdateAlarm) ExampleOutput() map[string]any {
	return exampleOutputUpdateAlarm.Value()
}

func (c *DeleteAlarm) ExampleOutput() map[string]any {
	return exampleOutputDeleteAlarm.Value()
}
