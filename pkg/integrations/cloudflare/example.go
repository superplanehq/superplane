package cloudflare

import (
	_ "embed"
	"sync"

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

var exampleOutputUpdateRedirectRuleOnce sync.Once
var exampleOutputUpdateRedirectRule map[string]any

var exampleOutputCreateDNSRecordOnce sync.Once
var exampleOutputCreateDNSRecord map[string]any

var exampleOutputCreateMonitorOnce sync.Once
var exampleOutputCreateMonitor map[string]any

var exampleOutputDeleteMonitorOnce sync.Once
var exampleOutputDeleteMonitor map[string]any

var exampleOutputGetMonitorOnce sync.Once
var exampleOutputGetMonitor map[string]any

var exampleOutputUpdateMonitorOnce sync.Once
var exampleOutputUpdateMonitor map[string]any

var exampleDataOnLoadBalancingHealthAlertOnce sync.Once
var exampleDataOnLoadBalancingHealthAlert map[string]any

func (c *CreateDNSRecord) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateDNSRecordOnce, exampleOutputCreateDNSRecordBytes, &exampleOutputCreateDNSRecord)
}

func (c *UpdateRedirectRule) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputUpdateRedirectRuleOnce, exampleOutputUpdateRedirectRuleBytes, &exampleOutputUpdateRedirectRule)
}

func (c *CreateMonitor) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateMonitorOnce, exampleOutputCreateMonitorBytes, &exampleOutputCreateMonitor)
}

func (c *DeleteMonitor) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputDeleteMonitorOnce, exampleOutputDeleteMonitorBytes, &exampleOutputDeleteMonitor)
}

func (c *GetMonitor) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetMonitorOnce, exampleOutputGetMonitorBytes, &exampleOutputGetMonitor)
}

func (c *UpdateMonitor) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputUpdateMonitorOnce, exampleOutputUpdateMonitorBytes, &exampleOutputUpdateMonitor)
}

func (t *OnLoadBalancingHealthAlert) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnLoadBalancingHealthAlertOnce, exampleDataOnLoadBalancingHealthAlertBytes, &exampleDataOnLoadBalancingHealthAlert)
}

//go:embed example_output_update_dns_record.json
var exampleOutputUpdateDNSRecordBytes []byte

var exampleOutputUpdateDNSRecordOnce sync.Once
var exampleOutputUpdateDNSRecord map[string]any

func (c *UpdateDNSRecord) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputUpdateDNSRecordOnce, exampleOutputUpdateDNSRecordBytes, &exampleOutputUpdateDNSRecord)
}

//go:embed example_output_delete_dns_record.json
var exampleOutputDeleteDNSRecordBytes []byte

var exampleOutputDeleteDNSRecordOnce sync.Once
var exampleOutputDeleteDNSRecord map[string]any

func (c *DeleteDNSRecord) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputDeleteDNSRecordOnce, exampleOutputDeleteDNSRecordBytes, &exampleOutputDeleteDNSRecord)
}

//go:embed example_output_create_kv_namespace.json
var exampleOutputCreateKVNamespaceBytes []byte

var exampleOutputCreateKVNamespaceOnce sync.Once
var exampleOutputCreateKVNamespace map[string]any

func (c *CreateKVNamespace) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateKVNamespaceOnce, exampleOutputCreateKVNamespaceBytes, &exampleOutputCreateKVNamespace)
}

//go:embed example_output_put_kv_value.json
var exampleOutputPutKVValueBytes []byte

var exampleOutputPutKVValueOnce sync.Once
var exampleOutputPutKVValue map[string]any

func (c *PutKVValue) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputPutKVValueOnce, exampleOutputPutKVValueBytes, &exampleOutputPutKVValue)
}

//go:embed example_output_get_kv_value.json
var exampleOutputGetKVValueBytes []byte

var exampleOutputGetKVValueOnce sync.Once
var exampleOutputGetKVValue map[string]any

func (c *GetKVValue) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetKVValueOnce, exampleOutputGetKVValueBytes, &exampleOutputGetKVValue)
}

//go:embed example_output_delete_kv_value.json
var exampleOutputDeleteKVValueBytes []byte

var exampleOutputDeleteKVValueOnce sync.Once
var exampleOutputDeleteKVValue map[string]any

func (c *DeleteKVValue) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputDeleteKVValueOnce, exampleOutputDeleteKVValueBytes, &exampleOutputDeleteKVValue)
}

//go:embed example_output_delete_kv_namespace.json
var exampleOutputDeleteKVNamespaceBytes []byte

var exampleOutputDeleteKVNamespaceOnce sync.Once
var exampleOutputDeleteKVNamespace map[string]any

func (c *DeleteKVNamespace) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputDeleteKVNamespaceOnce, exampleOutputDeleteKVNamespaceBytes, &exampleOutputDeleteKVNamespace)
}

//go:embed example_output_create_pool.json
var exampleOutputCreatePoolBytes []byte

var exampleOutputCreatePoolOnce sync.Once
var exampleOutputCreatePool map[string]any

func (c *CreatePool) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreatePoolOnce, exampleOutputCreatePoolBytes, &exampleOutputCreatePool)
}

//go:embed example_output_update_pool.json
var exampleOutputUpdatePoolBytes []byte

var exampleOutputUpdatePoolOnce sync.Once
var exampleOutputUpdatePool map[string]any

func (c *UpdatePool) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputUpdatePoolOnce, exampleOutputUpdatePoolBytes, &exampleOutputUpdatePool)
}

//go:embed example_output_get_pool.json
var exampleOutputGetPoolBytes []byte

var exampleOutputGetPoolOnce sync.Once
var exampleOutputGetPool map[string]any

func (c *GetPool) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetPoolOnce, exampleOutputGetPoolBytes, &exampleOutputGetPool)
}

//go:embed example_output_delete_pool.json
var exampleOutputDeletePoolBytes []byte

var exampleOutputDeletePoolOnce sync.Once
var exampleOutputDeletePool map[string]any

func (c *DeletePool) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputDeletePoolOnce, exampleOutputDeletePoolBytes, &exampleOutputDeletePool)
}

//go:embed example_output_purge_cache.json
var exampleOutputPurgeCacheBytes []byte

var exampleOutputPurgeCacheOnce sync.Once
var exampleOutputPurgeCache map[string]any

func (c *PurgeCache) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputPurgeCacheOnce, exampleOutputPurgeCacheBytes, &exampleOutputPurgeCache)
}

//go:embed example_output_order_certificate_pack.json
var exampleOutputOrderCertificatePackBytes []byte

var exampleOutputOrderCertificatePackOnce sync.Once
var exampleOutputOrderCertificatePack map[string]any

func (c *OrderCertificatePack) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputOrderCertificatePackOnce, exampleOutputOrderCertificatePackBytes, &exampleOutputOrderCertificatePack)
}

//go:embed example_output_delete_certificate_pack.json
var exampleOutputDeleteCertificatePackBytes []byte

var exampleOutputDeleteCertificatePackOnce sync.Once
var exampleOutputDeleteCertificatePack map[string]any

func (c *DeleteCertificatePack) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputDeleteCertificatePackOnce, exampleOutputDeleteCertificatePackBytes, &exampleOutputDeleteCertificatePack)
}

//go:embed example_output_create_load_balancer.json
var exampleOutputCreateLoadBalancerBytes []byte

var exampleOutputCreateLoadBalancerOnce sync.Once
var exampleOutputCreateLoadBalancer map[string]any

func (c *CreateLoadBalancer) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateLoadBalancerOnce, exampleOutputCreateLoadBalancerBytes, &exampleOutputCreateLoadBalancer)
}

//go:embed example_output_get_load_balancer.json
var exampleOutputGetLoadBalancerBytes []byte

var exampleOutputGetLoadBalancerOnce sync.Once
var exampleOutputGetLoadBalancer map[string]any

func (c *GetLoadBalancer) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetLoadBalancerOnce, exampleOutputGetLoadBalancerBytes, &exampleOutputGetLoadBalancer)
}

//go:embed example_output_update_load_balancer.json
var exampleOutputUpdateLoadBalancerBytes []byte

var exampleOutputUpdateLoadBalancerOnce sync.Once
var exampleOutputUpdateLoadBalancer map[string]any

func (c *UpdateLoadBalancer) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputUpdateLoadBalancerOnce, exampleOutputUpdateLoadBalancerBytes, &exampleOutputUpdateLoadBalancer)
}

//go:embed example_output_delete_load_balancer.json
var exampleOutputDeleteLoadBalancerBytes []byte

var exampleOutputDeleteLoadBalancerOnce sync.Once
var exampleOutputDeleteLoadBalancer map[string]any

func (c *DeleteLoadBalancer) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputDeleteLoadBalancerOnce, exampleOutputDeleteLoadBalancerBytes, &exampleOutputDeleteLoadBalancer)
}

//go:embed example_output_deploy_worker.json
var exampleOutputDeployWorkerBytes []byte

var exampleOutputDeployWorkerOnce sync.Once
var exampleOutputDeployWorker map[string]any

func (d *DeployWorker) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputDeployWorkerOnce, exampleOutputDeployWorkerBytes, &exampleOutputDeployWorker)
}

//go:embed example_output_get_worker.json
var exampleOutputGetWorkerBytes []byte

var exampleOutputGetWorkerOnce sync.Once
var exampleOutputGetWorker map[string]any

func (g *GetWorker) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetWorkerOnce, exampleOutputGetWorkerBytes, &exampleOutputGetWorker)
}

//go:embed example_output_delete_worker.json
var exampleOutputDeleteWorkerBytes []byte

var exampleOutputDeleteWorkerOnce sync.Once
var exampleOutputDeleteWorker map[string]any

func (d *DeleteWorker) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputDeleteWorkerOnce, exampleOutputDeleteWorkerBytes, &exampleOutputDeleteWorker)
}

//go:embed example_output_update_worker_route.json
var exampleOutputUpdateWorkerRouteBytes []byte

var exampleOutputUpdateWorkerRouteOnce sync.Once
var exampleOutputUpdateWorkerRoute map[string]any

func (u *UpdateWorkerRoute) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputUpdateWorkerRouteOnce, exampleOutputUpdateWorkerRouteBytes, &exampleOutputUpdateWorkerRoute)
}

//go:embed example_output_create_tunnel.json
var exampleOutputCreateTunnelBytes []byte

var exampleOutputCreateTunnelOnce sync.Once
var exampleOutputCreateTunnel map[string]any

func (c *CreateTunnel) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateTunnelOnce, exampleOutputCreateTunnelBytes, &exampleOutputCreateTunnel)
}

//go:embed example_output_get_tunnel.json
var exampleOutputGetTunnelBytes []byte

var exampleOutputGetTunnelOnce sync.Once
var exampleOutputGetTunnel map[string]any

func (c *GetTunnel) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetTunnelOnce, exampleOutputGetTunnelBytes, &exampleOutputGetTunnel)
}

//go:embed example_output_delete_tunnel.json
var exampleOutputDeleteTunnelBytes []byte

var exampleOutputDeleteTunnelOnce sync.Once
var exampleOutputDeleteTunnel map[string]any

func (c *DeleteTunnel) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputDeleteTunnelOnce, exampleOutputDeleteTunnelBytes, &exampleOutputDeleteTunnel)
}

//go:embed example_data_on_tunnel_health.json
var exampleDataOnTunnelHealthBytes []byte

var exampleDataOnTunnelHealthOnce sync.Once
var exampleDataOnTunnelHealth map[string]any

func (t *OnTunnelHealth) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnTunnelHealthOnce, exampleDataOnTunnelHealthBytes, &exampleDataOnTunnelHealth)
}
