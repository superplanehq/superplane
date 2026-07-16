package compute

import (
	_ "embed"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_create_vm.json
var exampleOutputCreateVMBytes []byte

//go:embed example_output_delete_vm_instance.json
var exampleOutputDeleteVMInstanceBytes []byte

//go:embed example_data_on_vm_instance.json
var exampleDataOnVMInstanceBytes []byte

//go:embed example_output_get_vm_instance.json
var exampleOutputGetVMInstanceBytes []byte

//go:embed example_output_manage_vm_instance_power.json
var exampleOutputManageVMInstancePowerBytes []byte

//go:embed example_output_update_vm_instance_type.json
var exampleOutputUpdateVMInstanceTypeBytes []byte

//go:embed example_output_get_vm_instance_metrics.json
var exampleOutputGetVMInstanceMetricsBytes []byte

//go:embed example_output_create_image.json
var exampleOutputCreateImageBytes []byte

//go:embed example_output_update_image.json
var exampleOutputUpdateImageBytes []byte

//go:embed example_output_delete_image.json
var exampleOutputDeleteImageBytes []byte

//go:embed example_output_create_static_ip.json
var exampleOutputCreateStaticIPBytes []byte

//go:embed example_output_delete_static_ip.json
var exampleOutputDeleteStaticIPBytes []byte

//go:embed example_output_manage_static_ip.json
var exampleOutputManageStaticIPBytes []byte

//go:embed example_output_create_load_balancer.json
var exampleOutputCreateLoadBalancerBytes []byte

//go:embed example_output_delete_load_balancer.json
var exampleOutputDeleteLoadBalancerBytes []byte

//go:embed example_output_create_firewall_rule.json
var exampleOutputCreateFirewallRuleBytes []byte

//go:embed example_output_update_firewall_rule.json
var exampleOutputUpdateFirewallRuleBytes []byte

//go:embed example_output_delete_firewall_rule.json
var exampleOutputDeleteFirewallRuleBytes []byte

var (
	exampleOutputCreateVM              = utils.NewEmbeddedJSON(exampleOutputCreateVMBytes)
	exampleOutputDeleteVMInstance      = utils.NewEmbeddedJSON(exampleOutputDeleteVMInstanceBytes)
	exampleDataOnVMInstance            = utils.NewEmbeddedJSON(exampleDataOnVMInstanceBytes)
	exampleOutputGetVMInstance         = utils.NewEmbeddedJSON(exampleOutputGetVMInstanceBytes)
	exampleOutputManageVMInstancePower = utils.NewEmbeddedJSON(exampleOutputManageVMInstancePowerBytes)
	exampleOutputUpdateVMInstanceType  = utils.NewEmbeddedJSON(exampleOutputUpdateVMInstanceTypeBytes)
	exampleOutputGetVMInstanceMetrics  = utils.NewEmbeddedJSON(exampleOutputGetVMInstanceMetricsBytes)
	exampleOutputCreateImage           = utils.NewEmbeddedJSON(exampleOutputCreateImageBytes)
	exampleOutputUpdateImage           = utils.NewEmbeddedJSON(exampleOutputUpdateImageBytes)
	exampleOutputDeleteImage           = utils.NewEmbeddedJSON(exampleOutputDeleteImageBytes)
	exampleOutputCreateStaticIP        = utils.NewEmbeddedJSON(exampleOutputCreateStaticIPBytes)
	exampleOutputDeleteStaticIP        = utils.NewEmbeddedJSON(exampleOutputDeleteStaticIPBytes)
	exampleOutputManageStaticIP        = utils.NewEmbeddedJSON(exampleOutputManageStaticIPBytes)
	exampleOutputCreateLoadBalancer    = utils.NewEmbeddedJSON(exampleOutputCreateLoadBalancerBytes)
	exampleOutputDeleteLoadBalancer    = utils.NewEmbeddedJSON(exampleOutputDeleteLoadBalancerBytes)
	exampleOutputCreateFirewallRule    = utils.NewEmbeddedJSON(exampleOutputCreateFirewallRuleBytes)
	exampleOutputUpdateFirewallRule    = utils.NewEmbeddedJSON(exampleOutputUpdateFirewallRuleBytes)
	exampleOutputDeleteFirewallRule    = utils.NewEmbeddedJSON(exampleOutputDeleteFirewallRuleBytes)
)

func (c *CreateVM) ExampleOutput() map[string]any {
	return exampleOutputCreateVM.Value()
}

func (d *DeleteVMInstance) ExampleOutput() map[string]any {
	return exampleOutputDeleteVMInstance.Value()
}

func (g *GetVMInstance) ExampleOutput() map[string]any {
	return exampleOutputGetVMInstance.Value()
}

func (t *OnVMInstance) ExampleData() map[string]any {
	return exampleDataOnVMInstance.Value()
}

func (m *ManageVMInstancePower) ExampleOutput() map[string]any {
	return exampleOutputManageVMInstancePower.Value()
}

func (u *UpdateVMInstanceType) ExampleOutput() map[string]any {
	return exampleOutputUpdateVMInstanceType.Value()
}

func (g *GetVMInstanceMetrics) ExampleOutput() map[string]any {
	return exampleOutputGetVMInstanceMetrics.Value()
}

func (c *CreateImage) ExampleOutput() map[string]any {
	return exampleOutputCreateImage.Value()
}

func (u *UpdateImage) ExampleOutput() map[string]any {
	return exampleOutputUpdateImage.Value()
}

func (d *DeleteImage) ExampleOutput() map[string]any {
	return exampleOutputDeleteImage.Value()
}

func (c *CreateStaticIP) ExampleOutput() map[string]any {
	return exampleOutputCreateStaticIP.Value()
}

func (d *DeleteStaticIP) ExampleOutput() map[string]any {
	return exampleOutputDeleteStaticIP.Value()
}

func (m *ManageStaticIP) ExampleOutput() map[string]any {
	return exampleOutputManageStaticIP.Value()
}

func (c *CreateLoadBalancer) ExampleOutput() map[string]any {
	return exampleOutputCreateLoadBalancer.Value()
}

func (d *DeleteLoadBalancer) ExampleOutput() map[string]any {
	return exampleOutputDeleteLoadBalancer.Value()
}

func (c *CreateFirewall) ExampleOutput() map[string]any {
	return exampleOutputCreateFirewallRule.Value()
}

func (u *UpdateFirewall) ExampleOutput() map[string]any {
	return exampleOutputUpdateFirewallRule.Value()
}

func (d *DeleteFirewall) ExampleOutput() map[string]any {
	return exampleOutputDeleteFirewallRule.Value()
}
