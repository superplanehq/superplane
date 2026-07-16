package cloudflare

import (
	_ "embed"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_update_redirect_rule.json
var exampleOutputUpdateRedirectRuleBytes []byte

//go:embed example_output_create_dns_record.json
var exampleOutputCreateDNSRecordBytes []byte

//go:embed example_output_create_monitor.json
var exampleOutputCreateMonitorBytes []byte

//go:embed example_output_delete_monitor.json
var exampleOutputDeleteMonitorBytes []byte

//go:embed example_output_get_monitor.json
var exampleOutputGetMonitorBytes []byte

//go:embed example_output_update_monitor.json
var exampleOutputUpdateMonitorBytes []byte

//go:embed example_data_on_load_balancing_health_alert.json
var exampleDataOnLoadBalancingHealthAlertBytes []byte
var exampleOutputUpdateRedirectRule = utils.NewEmbeddedJSON(exampleOutputUpdateRedirectRuleBytes)
var exampleOutputCreateDNSRecord = utils.NewEmbeddedJSON(exampleOutputCreateDNSRecordBytes)
var exampleOutputCreateMonitor = utils.NewEmbeddedJSON(exampleOutputCreateMonitorBytes)
var exampleOutputDeleteMonitor = utils.NewEmbeddedJSON(exampleOutputDeleteMonitorBytes)
var exampleOutputGetMonitor = utils.NewEmbeddedJSON(exampleOutputGetMonitorBytes)
var exampleOutputUpdateMonitor = utils.NewEmbeddedJSON(exampleOutputUpdateMonitorBytes)
var exampleDataOnLoadBalancingHealthAlert = utils.NewEmbeddedJSON(exampleDataOnLoadBalancingHealthAlertBytes)

func (c *CreateDNSRecord) ExampleOutput() map[string]any {
	return exampleOutputCreateDNSRecord.Value()
}

func (c *UpdateRedirectRule) ExampleOutput() map[string]any {
	return exampleOutputUpdateRedirectRule.Value()
}

func (c *CreateMonitor) ExampleOutput() map[string]any {
	return exampleOutputCreateMonitor.Value()
}

func (c *DeleteMonitor) ExampleOutput() map[string]any {
	return exampleOutputDeleteMonitor.Value()
}

func (c *GetMonitor) ExampleOutput() map[string]any {
	return exampleOutputGetMonitor.Value()
}

func (c *UpdateMonitor) ExampleOutput() map[string]any {
	return exampleOutputUpdateMonitor.Value()
}

func (t *OnLoadBalancingHealthAlert) ExampleData() map[string]any {
	return exampleDataOnLoadBalancingHealthAlert.Value()
}

//go:embed example_output_update_dns_record.json
var exampleOutputUpdateDNSRecordBytes []byte
var exampleOutputUpdateDNSRecord = utils.NewEmbeddedJSON(exampleOutputUpdateDNSRecordBytes)

func (c *UpdateDNSRecord) ExampleOutput() map[string]any {
	return exampleOutputUpdateDNSRecord.Value()
}

//go:embed example_output_delete_dns_record.json
var exampleOutputDeleteDNSRecordBytes []byte
var exampleOutputDeleteDNSRecord = utils.NewEmbeddedJSON(exampleOutputDeleteDNSRecordBytes)

func (c *DeleteDNSRecord) ExampleOutput() map[string]any {
	return exampleOutputDeleteDNSRecord.Value()
}

//go:embed example_output_create_kv_namespace.json
var exampleOutputCreateKVNamespaceBytes []byte
var exampleOutputCreateKVNamespace = utils.NewEmbeddedJSON(exampleOutputCreateKVNamespaceBytes)

func (c *CreateKVNamespace) ExampleOutput() map[string]any {
	return exampleOutputCreateKVNamespace.Value()
}

//go:embed example_output_put_kv_value.json
var exampleOutputPutKVValueBytes []byte
var exampleOutputPutKVValue = utils.NewEmbeddedJSON(exampleOutputPutKVValueBytes)

func (c *PutKVValue) ExampleOutput() map[string]any {
	return exampleOutputPutKVValue.Value()
}

//go:embed example_output_get_kv_value.json
var exampleOutputGetKVValueBytes []byte
var exampleOutputGetKVValue = utils.NewEmbeddedJSON(exampleOutputGetKVValueBytes)

func (c *GetKVValue) ExampleOutput() map[string]any {
	return exampleOutputGetKVValue.Value()
}

//go:embed example_output_delete_kv_value.json
var exampleOutputDeleteKVValueBytes []byte
var exampleOutputDeleteKVValue = utils.NewEmbeddedJSON(exampleOutputDeleteKVValueBytes)

func (c *DeleteKVValue) ExampleOutput() map[string]any {
	return exampleOutputDeleteKVValue.Value()
}

//go:embed example_output_delete_kv_namespace.json
var exampleOutputDeleteKVNamespaceBytes []byte
var exampleOutputDeleteKVNamespace = utils.NewEmbeddedJSON(exampleOutputDeleteKVNamespaceBytes)

func (c *DeleteKVNamespace) ExampleOutput() map[string]any {
	return exampleOutputDeleteKVNamespace.Value()
}

//go:embed example_output_create_pool.json
var exampleOutputCreatePoolBytes []byte
var exampleOutputCreatePool = utils.NewEmbeddedJSON(exampleOutputCreatePoolBytes)

func (c *CreatePool) ExampleOutput() map[string]any {
	return exampleOutputCreatePool.Value()
}

//go:embed example_output_update_pool.json
var exampleOutputUpdatePoolBytes []byte
var exampleOutputUpdatePool = utils.NewEmbeddedJSON(exampleOutputUpdatePoolBytes)

func (c *UpdatePool) ExampleOutput() map[string]any {
	return exampleOutputUpdatePool.Value()
}

//go:embed example_output_get_pool.json
var exampleOutputGetPoolBytes []byte
var exampleOutputGetPool = utils.NewEmbeddedJSON(exampleOutputGetPoolBytes)

func (c *GetPool) ExampleOutput() map[string]any {
	return exampleOutputGetPool.Value()
}

//go:embed example_output_delete_pool.json
var exampleOutputDeletePoolBytes []byte
var exampleOutputDeletePool = utils.NewEmbeddedJSON(exampleOutputDeletePoolBytes)

func (c *DeletePool) ExampleOutput() map[string]any {
	return exampleOutputDeletePool.Value()
}

//go:embed example_output_purge_cache.json
var exampleOutputPurgeCacheBytes []byte
var exampleOutputPurgeCache = utils.NewEmbeddedJSON(exampleOutputPurgeCacheBytes)

func (c *PurgeCache) ExampleOutput() map[string]any {
	return exampleOutputPurgeCache.Value()
}

//go:embed example_output_order_certificate_pack.json
var exampleOutputOrderCertificatePackBytes []byte
var exampleOutputOrderCertificatePack = utils.NewEmbeddedJSON(exampleOutputOrderCertificatePackBytes)

func (c *OrderCertificatePack) ExampleOutput() map[string]any {
	return exampleOutputOrderCertificatePack.Value()
}

//go:embed example_output_delete_certificate_pack.json
var exampleOutputDeleteCertificatePackBytes []byte
var exampleOutputDeleteCertificatePack = utils.NewEmbeddedJSON(exampleOutputDeleteCertificatePackBytes)

func (c *DeleteCertificatePack) ExampleOutput() map[string]any {
	return exampleOutputDeleteCertificatePack.Value()
}

//go:embed example_output_create_load_balancer.json
var exampleOutputCreateLoadBalancerBytes []byte
var exampleOutputCreateLoadBalancer = utils.NewEmbeddedJSON(exampleOutputCreateLoadBalancerBytes)

func (c *CreateLoadBalancer) ExampleOutput() map[string]any {
	return exampleOutputCreateLoadBalancer.Value()
}

//go:embed example_output_get_load_balancer.json
var exampleOutputGetLoadBalancerBytes []byte
var exampleOutputGetLoadBalancer = utils.NewEmbeddedJSON(exampleOutputGetLoadBalancerBytes)

func (c *GetLoadBalancer) ExampleOutput() map[string]any {
	return exampleOutputGetLoadBalancer.Value()
}

//go:embed example_output_update_load_balancer.json
var exampleOutputUpdateLoadBalancerBytes []byte
var exampleOutputUpdateLoadBalancer = utils.NewEmbeddedJSON(exampleOutputUpdateLoadBalancerBytes)

func (c *UpdateLoadBalancer) ExampleOutput() map[string]any {
	return exampleOutputUpdateLoadBalancer.Value()
}

//go:embed example_output_delete_load_balancer.json
var exampleOutputDeleteLoadBalancerBytes []byte
var exampleOutputDeleteLoadBalancer = utils.NewEmbeddedJSON(exampleOutputDeleteLoadBalancerBytes)

func (c *DeleteLoadBalancer) ExampleOutput() map[string]any {
	return exampleOutputDeleteLoadBalancer.Value()
}

//go:embed example_output_deploy_worker.json
var exampleOutputDeployWorkerBytes []byte
var exampleOutputDeployWorker = utils.NewEmbeddedJSON(exampleOutputDeployWorkerBytes)

func (d *DeployWorker) ExampleOutput() map[string]any {
	return exampleOutputDeployWorker.Value()
}

//go:embed example_output_get_worker.json
var exampleOutputGetWorkerBytes []byte
var exampleOutputGetWorker = utils.NewEmbeddedJSON(exampleOutputGetWorkerBytes)

func (g *GetWorker) ExampleOutput() map[string]any {
	return exampleOutputGetWorker.Value()
}

//go:embed example_output_delete_worker.json
var exampleOutputDeleteWorkerBytes []byte
var exampleOutputDeleteWorker = utils.NewEmbeddedJSON(exampleOutputDeleteWorkerBytes)

func (d *DeleteWorker) ExampleOutput() map[string]any {
	return exampleOutputDeleteWorker.Value()
}

//go:embed example_output_update_worker_route.json
var exampleOutputUpdateWorkerRouteBytes []byte
var exampleOutputUpdateWorkerRoute = utils.NewEmbeddedJSON(exampleOutputUpdateWorkerRouteBytes)

func (u *UpdateWorkerRoute) ExampleOutput() map[string]any {
	return exampleOutputUpdateWorkerRoute.Value()
}

//go:embed example_output_create_tunnel.json
var exampleOutputCreateTunnelBytes []byte
var exampleOutputCreateTunnel = utils.NewEmbeddedJSON(exampleOutputCreateTunnelBytes)

func (c *CreateTunnel) ExampleOutput() map[string]any {
	return exampleOutputCreateTunnel.Value()
}

//go:embed example_output_get_tunnel.json
var exampleOutputGetTunnelBytes []byte
var exampleOutputGetTunnel = utils.NewEmbeddedJSON(exampleOutputGetTunnelBytes)

func (c *GetTunnel) ExampleOutput() map[string]any {
	return exampleOutputGetTunnel.Value()
}

//go:embed example_output_delete_tunnel.json
var exampleOutputDeleteTunnelBytes []byte
var exampleOutputDeleteTunnel = utils.NewEmbeddedJSON(exampleOutputDeleteTunnelBytes)

func (c *DeleteTunnel) ExampleOutput() map[string]any {
	return exampleOutputDeleteTunnel.Value()
}

//go:embed example_data_on_tunnel_health.json
var exampleDataOnTunnelHealthBytes []byte
var exampleDataOnTunnelHealth = utils.NewEmbeddedJSON(exampleDataOnTunnelHealthBytes)

func (t *OnTunnelHealth) ExampleData() map[string]any {
	return exampleDataOnTunnelHealth.Value()
}
