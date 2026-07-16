package digitalocean

import (
	_ "embed"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_create_droplet.json
var exampleOutputCreateDropletBytes []byte
var exampleOutputCreateDroplet = utils.NewEmbeddedJSON(exampleOutputCreateDropletBytes)

func (c *CreateDroplet) ExampleOutput() map[string]any {
	return exampleOutputCreateDroplet.Value()
}

//go:embed example_output_get_droplet.json
var exampleOutputGetDropletBytes []byte
var exampleOutputGetDroplet = utils.NewEmbeddedJSON(exampleOutputGetDropletBytes)

func (g *GetDroplet) ExampleOutput() map[string]any {
	return exampleOutputGetDroplet.Value()
}

//go:embed example_output_delete_droplet.json
var exampleOutputDeleteDropletBytes []byte
var exampleOutputDeleteDroplet = utils.NewEmbeddedJSON(exampleOutputDeleteDropletBytes)

func (d *DeleteDroplet) ExampleOutput() map[string]any {
	return exampleOutputDeleteDroplet.Value()
}

//go:embed example_output_manage_droplet_power.json
var exampleOutputManageDropletPowerBytes []byte
var exampleOutputManageDropletPower = utils.NewEmbeddedJSON(exampleOutputManageDropletPowerBytes)

func (m *ManageDropletPower) ExampleOutput() map[string]any {
	return exampleOutputManageDropletPower.Value()
}

//go:embed example_output_create_snapshot.json
var exampleOutputCreateSnapshotBytes []byte
var exampleOutputCreateSnapshot = utils.NewEmbeddedJSON(exampleOutputCreateSnapshotBytes)

func (c *CreateSnapshot) ExampleOutput() map[string]any {
	return exampleOutputCreateSnapshot.Value()
}

//go:embed example_output_delete_snapshot.json
var exampleOutputDeleteSnapshotBytes []byte
var exampleOutputDeleteSnapshot = utils.NewEmbeddedJSON(exampleOutputDeleteSnapshotBytes)

func (c *DeleteSnapshot) ExampleOutput() map[string]any {
	return exampleOutputDeleteSnapshot.Value()
}

//go:embed example_output_create_dns_record.json
var exampleOutputCreateDNSRecordBytes []byte
var exampleOutputCreateDNSRecord = utils.NewEmbeddedJSON(exampleOutputCreateDNSRecordBytes)

func (c *CreateDNSRecord) ExampleOutput() map[string]any {
	return exampleOutputCreateDNSRecord.Value()
}

//go:embed example_output_delete_dns_record.json
var exampleOutputDeleteDNSRecordBytes []byte
var exampleOutputDeleteDNSRecord = utils.NewEmbeddedJSON(exampleOutputDeleteDNSRecordBytes)

func (d *DeleteDNSRecord) ExampleOutput() map[string]any {
	return exampleOutputDeleteDNSRecord.Value()
}

//go:embed example_output_upsert_dns_record.json
var exampleOutputUpsertDNSRecordBytes []byte
var exampleOutputUpsertDNSRecord = utils.NewEmbeddedJSON(exampleOutputUpsertDNSRecordBytes)

func (u *UpsertDNSRecord) ExampleOutput() map[string]any {
	return exampleOutputUpsertDNSRecord.Value()
}

//go:embed example_output_create_load_balancer.json
var exampleOutputCreateLoadBalancerBytes []byte
var exampleOutputCreateLoadBalancer = utils.NewEmbeddedJSON(exampleOutputCreateLoadBalancerBytes)

func (c *CreateLoadBalancer) ExampleOutput() map[string]any {
	return exampleOutputCreateLoadBalancer.Value()
}

//go:embed example_output_delete_load_balancer.json
var exampleOutputDeleteLoadBalancerBytes []byte
var exampleOutputDeleteLoadBalancer = utils.NewEmbeddedJSON(exampleOutputDeleteLoadBalancerBytes)

func (d *DeleteLoadBalancer) ExampleOutput() map[string]any {
	return exampleOutputDeleteLoadBalancer.Value()
}

//go:embed example_output_assign_reserved_ip.json
var exampleOutputAssignReservedIPBytes []byte
var exampleOutputAssignReservedIP = utils.NewEmbeddedJSON(exampleOutputAssignReservedIPBytes)

func (a *AssignReservedIP) ExampleOutput() map[string]any {
	return exampleOutputAssignReservedIP.Value()
}

//go:embed example_output_create_alert_policy.json
var exampleOutputCreateAlertPolicyBytes []byte
var exampleOutputCreateAlertPolicy = utils.NewEmbeddedJSON(exampleOutputCreateAlertPolicyBytes)

func (c *CreateAlertPolicy) ExampleOutput() map[string]any {
	return exampleOutputCreateAlertPolicy.Value()
}

//go:embed example_output_get_alert_policy.json
var exampleOutputGetAlertPolicyBytes []byte
var exampleOutputGetAlertPolicy = utils.NewEmbeddedJSON(exampleOutputGetAlertPolicyBytes)

func (g *GetAlertPolicy) ExampleOutput() map[string]any {
	return exampleOutputGetAlertPolicy.Value()
}

//go:embed example_output_delete_alert_policy.json
var exampleOutputDeleteAlertPolicyBytes []byte
var exampleOutputDeleteAlertPolicy = utils.NewEmbeddedJSON(exampleOutputDeleteAlertPolicyBytes)

func (d *DeleteAlertPolicy) ExampleOutput() map[string]any {
	return exampleOutputDeleteAlertPolicy.Value()
}

//go:embed example_output_update_alert_policy.json
var exampleOutputUpdateAlertPolicyBytes []byte
var exampleOutputUpdateAlertPolicy = utils.NewEmbeddedJSON(exampleOutputUpdateAlertPolicyBytes)

func (u *UpdateAlertPolicy) ExampleOutput() map[string]any {
	return exampleOutputUpdateAlertPolicy.Value()
}

//go:embed example_output_get_droplet_metrics.json
var exampleOutputGetDropletMetricsBytes []byte
var exampleOutputGetDropletMetrics = utils.NewEmbeddedJSON(exampleOutputGetDropletMetricsBytes)

func (g *GetDropletMetrics) ExampleOutput() map[string]any {
	return exampleOutputGetDropletMetrics.Value()
}

//go:embed example_output_get_database.json
var exampleOutputGetDatabaseBytes []byte
var exampleOutputGetDatabase = utils.NewEmbeddedJSON(exampleOutputGetDatabaseBytes)

func (g *GetDatabase) ExampleOutput() map[string]any {
	return exampleOutputGetDatabase.Value()
}

//go:embed example_output_get_cluster_configuration.json
var exampleOutputGetClusterConfigurationBytes []byte
var exampleOutputGetClusterConfiguration = utils.NewEmbeddedJSON(exampleOutputGetClusterConfigurationBytes)

func (g *GetClusterConfiguration) ExampleOutput() map[string]any {
	return exampleOutputGetClusterConfiguration.Value()
}

//go:embed example_output_get_object.json
var exampleOutputGetObjectBytes []byte
var exampleOutputGetObject = utils.NewEmbeddedJSON(exampleOutputGetObjectBytes)

func (g *GetObject) ExampleOutput() map[string]any {
	return exampleOutputGetObject.Value()
}

//go:embed example_output_copy_object.json
var exampleOutputCopyObjectBytes []byte
var exampleOutputCopyObject = utils.NewEmbeddedJSON(exampleOutputCopyObjectBytes)

func (c *CopyObject) ExampleOutput() map[string]any {
	return exampleOutputCopyObject.Value()
}

//go:embed example_output_delete_object.json
var exampleOutputDeleteObjectBytes []byte
var exampleOutputDeleteObject = utils.NewEmbeddedJSON(exampleOutputDeleteObjectBytes)

func (d *DeleteObject) ExampleOutput() map[string]any {
	return exampleOutputDeleteObject.Value()
}

//go:embed example_output_put_object.json
var exampleOutputPutObjectBytes []byte
var exampleOutputPutObject = utils.NewEmbeddedJSON(exampleOutputPutObjectBytes)

func (p *PutObject) ExampleOutput() map[string]any {
	return exampleOutputPutObject.Value()
}

//go:embed example_output_create_app.json
var exampleOutputCreateAppBytes []byte
var exampleOutputCreateApp = utils.NewEmbeddedJSON(exampleOutputCreateAppBytes)

func (c *CreateApp) ExampleOutput() map[string]any {
	return exampleOutputCreateApp.Value()
}

//go:embed example_output_create_database.json
var exampleOutputCreateDatabaseBytes []byte
var exampleOutputCreateDatabase = utils.NewEmbeddedJSON(exampleOutputCreateDatabaseBytes)

func (c *CreateDatabase) ExampleOutput() map[string]any {
	return exampleOutputCreateDatabase.Value()
}

//go:embed example_output_create_database_cluster.json
var exampleOutputCreateDatabaseClusterBytes []byte
var exampleOutputCreateDatabaseCluster = utils.NewEmbeddedJSON(exampleOutputCreateDatabaseClusterBytes)

func (c *CreateDatabaseCluster) ExampleOutput() map[string]any {
	return exampleOutputCreateDatabaseCluster.Value()
}

//go:embed example_output_get_app.json
var exampleOutputGetAppBytes []byte
var exampleOutputGetApp = utils.NewEmbeddedJSON(exampleOutputGetAppBytes)

func (g *GetApp) ExampleOutput() map[string]any {
	return exampleOutputGetApp.Value()
}

//go:embed example_output_get_database_cluster.json
var exampleOutputGetDatabaseClusterBytes []byte
var exampleOutputGetDatabaseCluster = utils.NewEmbeddedJSON(exampleOutputGetDatabaseClusterBytes)

func (g *GetDatabaseCluster) ExampleOutput() map[string]any {
	return exampleOutputGetDatabaseCluster.Value()
}

//go:embed example_output_delete_app.json
var exampleOutputDeleteAppBytes []byte
var exampleOutputDeleteApp = utils.NewEmbeddedJSON(exampleOutputDeleteAppBytes)

func (d *DeleteApp) ExampleOutput() map[string]any {
	return exampleOutputDeleteApp.Value()
}

//go:embed example_output_delete_database.json
var exampleOutputDeleteDatabaseBytes []byte
var exampleOutputDeleteDatabase = utils.NewEmbeddedJSON(exampleOutputDeleteDatabaseBytes)

func (d *DeleteDatabase) ExampleOutput() map[string]any {
	return exampleOutputDeleteDatabase.Value()
}

//go:embed example_output_update_app.json
var exampleOutputUpdateAppBytes []byte
var exampleOutputUpdateApp = utils.NewEmbeddedJSON(exampleOutputUpdateAppBytes)

func (u *UpdateApp) ExampleOutput() map[string]any {
	return exampleOutputUpdateApp.Value()
}

//go:embed example_output_get_knowledge_base.json
var exampleOutputGetKnowledgeBaseBytes []byte
var exampleOutputGetKnowledgeBase = utils.NewEmbeddedJSON(exampleOutputGetKnowledgeBaseBytes)

func (g *GetKnowledgeBase) ExampleOutput() map[string]any {
	return exampleOutputGetKnowledgeBase.Value()
}

//go:embed example_output_add_data_source.json
var exampleOutputAddDataSourceBytes []byte
var exampleOutputAddDataSource = utils.NewEmbeddedJSON(exampleOutputAddDataSourceBytes)

func (a *AddDataSource) ExampleOutput() map[string]any {
	return exampleOutputAddDataSource.Value()
}

//go:embed example_output_delete_data_source.json
var exampleOutputDeleteDataSourceBytes []byte
var exampleOutputDeleteDataSource = utils.NewEmbeddedJSON(exampleOutputDeleteDataSourceBytes)

func (d *DeleteDataSource) ExampleOutput() map[string]any {
	return exampleOutputDeleteDataSource.Value()
}

//go:embed example_output_index_knowledge_base.json
var exampleOutputIndexKnowledgeBaseBytes []byte
var exampleOutputIndexKnowledgeBase = utils.NewEmbeddedJSON(exampleOutputIndexKnowledgeBaseBytes)

func (i *IndexKnowledgeBase) ExampleOutput() map[string]any {
	return exampleOutputIndexKnowledgeBase.Value()
}

//go:embed example_output_create_knowledge_base.json
var exampleOutputCreateKnowledgeBaseBytes []byte
var exampleOutputCreateKnowledgeBase = utils.NewEmbeddedJSON(exampleOutputCreateKnowledgeBaseBytes)

func (c *CreateKnowledgeBase) ExampleOutput() map[string]any {
	return exampleOutputCreateKnowledgeBase.Value()
}

//go:embed example_output_attach_knowledge_base.json
var exampleOutputAttachKnowledgeBaseBytes []byte
var exampleOutputAttachKnowledgeBase = utils.NewEmbeddedJSON(exampleOutputAttachKnowledgeBaseBytes)

func (a *AttachKnowledgeBase) ExampleOutput() map[string]any {
	return exampleOutputAttachKnowledgeBase.Value()
}

//go:embed example_output_detach_knowledge_base.json
var exampleOutputDetachKnowledgeBaseBytes []byte
var exampleOutputDetachKnowledgeBase = utils.NewEmbeddedJSON(exampleOutputDetachKnowledgeBaseBytes)

func (d *DetachKnowledgeBase) ExampleOutput() map[string]any {
	return exampleOutputDetachKnowledgeBase.Value()
}

//go:embed example_output_delete_knowledge_base.json
var exampleOutputDeleteKnowledgeBaseBytes []byte
var exampleOutputDeleteKnowledgeBase = utils.NewEmbeddedJSON(exampleOutputDeleteKnowledgeBaseBytes)

func (d *DeleteKnowledgeBase) ExampleOutput() map[string]any {
	return exampleOutputDeleteKnowledgeBase.Value()
}

//go:embed example_output_run_evaluation.json
var exampleOutputRunEvaluationBytes []byte
var exampleOutputRunEvaluation = utils.NewEmbeddedJSON(exampleOutputRunEvaluationBytes)

func (r *RunEvaluation) ExampleOutput() map[string]any {
	return exampleOutputRunEvaluation.Value()
}

//go:embed example_output_create_gpu_droplet.json
var exampleOutputCreateGPUDropletBytes []byte
var exampleOutputCreateGPUDroplet = utils.NewEmbeddedJSON(exampleOutputCreateGPUDropletBytes)

func (c *CreateGPUDroplet) ExampleOutput() map[string]any {
	return exampleOutputCreateGPUDroplet.Value()
}

//go:embed example_output_get_gpu_droplet.json
var exampleOutputGetGPUDropletBytes []byte
var exampleOutputGetGPUDroplet = utils.NewEmbeddedJSON(exampleOutputGetGPUDropletBytes)

func (g *GetGPUDroplet) ExampleOutput() map[string]any {
	return exampleOutputGetGPUDroplet.Value()
}

//go:embed example_output_update_gpu_droplet.json
var exampleOutputUpdateGPUDropletBytes []byte
var exampleOutputUpdateGPUDroplet = utils.NewEmbeddedJSON(exampleOutputUpdateGPUDropletBytes)

func (u *UpdateGPUDroplet) ExampleOutput() map[string]any {
	return exampleOutputUpdateGPUDroplet.Value()
}

//go:embed example_output_delete_gpu_droplet.json
var exampleOutputDeleteGPUDropletBytes []byte
var exampleOutputDeleteGPUDroplet = utils.NewEmbeddedJSON(exampleOutputDeleteGPUDropletBytes)

func (d *DeleteGPUDroplet) ExampleOutput() map[string]any {
	return exampleOutputDeleteGPUDroplet.Value()
}
