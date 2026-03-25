import { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { createDropletMapper } from "./create_droplet";
import { getDropletMapper } from "./get_droplet";
import { deleteDropletMapper } from "./delete_droplet";
import { manageDropletPowerMapper, MANAGE_DROPLET_POWER_STATE_REGISTRY } from "./manage_droplet_power";
import { createSnapshotMapper } from "./create_snapshot";
import { deleteSnapshotMapper } from "./delete_snapshot";
import { createDNSRecordMapper } from "./create_dns_record";
import { deleteDNSRecordMapper } from "./delete_dns_record";
import { upsertDNSRecordMapper } from "./upsert_dns_record";
import { assignReservedIPMapper, ASSIGN_RESERVED_IP_STATE_REGISTRY } from "./assign_reserved_ip";
import { createLoadBalancerMapper } from "./create_load_balancer";
import { deleteLoadBalancerMapper } from "./delete_load_balancer";
import { createAlertPolicyMapper } from "./create_alert_policy";
import { getAlertPolicyMapper } from "./get_alert_policy";
import { deleteAlertPolicyMapper } from "./delete_alert_policy";
import { updateAlertPolicyMapper } from "./update_alert_policy";
import { getDropletMetricsMapper } from "./get_droplet_metrics";
import { getObjectMapper, GET_OBJECT_STATE_REGISTRY } from "./get_object";
import { createAppMapper } from "./create_app";
import { deleteAppMapper } from "./delete_app";
import { updateAppMapper } from "./update_app";
import { buildActionStateRegistry } from "../utils";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  createDroplet: createDropletMapper,
  getDroplet: getDropletMapper,
  deleteDroplet: deleteDropletMapper,
  manageDropletPower: manageDropletPowerMapper,
  createSnapshot: createSnapshotMapper,
  deleteSnapshot: deleteSnapshotMapper,
  createDNSRecord: createDNSRecordMapper,
  deleteDNSRecord: deleteDNSRecordMapper,
  upsertDNSRecord: upsertDNSRecordMapper,
  assignReservedIP: assignReservedIPMapper,
  createLoadBalancer: createLoadBalancerMapper,
  deleteLoadBalancer: deleteLoadBalancerMapper,
  createAlertPolicy: createAlertPolicyMapper,
  getAlertPolicy: getAlertPolicyMapper,
  deleteAlertPolicy: deleteAlertPolicyMapper,
  updateAlertPolicy: updateAlertPolicyMapper,
  getDropletMetrics: getDropletMetricsMapper,
  getObject: getObjectMapper,
  createApp: createAppMapper,
  deleteApp: deleteAppMapper,
  updateApp: updateAppMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  createDroplet: buildActionStateRegistry("created"),
  getDroplet: buildActionStateRegistry("fetched"),
  deleteDroplet: buildActionStateRegistry("deleted"),
  manageDropletPower: MANAGE_DROPLET_POWER_STATE_REGISTRY,
  createSnapshot: buildActionStateRegistry("created"),
  deleteSnapshot: buildActionStateRegistry("deleted"),
  createDNSRecord: buildActionStateRegistry("created"),
  deleteDNSRecord: buildActionStateRegistry("deleted"),
  upsertDNSRecord: buildActionStateRegistry("upserted"),
  assignReservedIP: ASSIGN_RESERVED_IP_STATE_REGISTRY,
  createLoadBalancer: buildActionStateRegistry("created"),
  deleteLoadBalancer: buildActionStateRegistry("deleted"),
  createAlertPolicy: buildActionStateRegistry("created"),
  getAlertPolicy: buildActionStateRegistry("fetched"),
  deleteAlertPolicy: buildActionStateRegistry("deleted"),
  updateAlertPolicy: buildActionStateRegistry("updated"),
  getDropletMetrics: buildActionStateRegistry("fetched"),
  getObject: GET_OBJECT_STATE_REGISTRY,
  createApp: buildActionStateRegistry("created"),
  deleteApp: buildActionStateRegistry("deleted"),
  updateApp: buildActionStateRegistry("updated"),
};
