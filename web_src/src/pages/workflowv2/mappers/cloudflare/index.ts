import type { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { baseMapper } from "./base";
import { buildActionStateRegistry } from "../utils";
import { onLoadBalancingHealthAlertTriggerRenderer } from "./on_load_balancing_health_alert";
import { createMonitorMapper } from "./create_monitor";
import { deleteMonitorMapper } from "./delete_monitor";
import { originRuleMapper } from "./origin_rule";
import { createKVNamespaceMapper } from "./create_kv_namespace";
import { putKVValueMapper } from "./put_kv_value";
import { getKVValueMapper } from "./get_kv_value";
import { deleteKVValueMapper } from "./delete_kv_value";
import { deleteKVNamespaceMapper } from "./delete_kv_namespace";
import { createPoolMapper } from "./create_pool";
import { getPoolMapper } from "./get_pool";
import { deletePoolMapper } from "./delete_pool";
import { updatePoolMapper } from "./update_pool";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  createDnsRecord: baseMapper,
  createMonitor: createMonitorMapper,
  createOriginRule: originRuleMapper,
  updateDNSRecord: baseMapper,
  deleteDnsRecord: baseMapper,
  deleteMonitor: deleteMonitorMapper,
  updateRedirectRule: baseMapper,
  updateOriginRule: originRuleMapper,
  deleteOriginRule: originRuleMapper,
  createKVNamespace: createKVNamespaceMapper,
  putKVValue: putKVValueMapper,
  getKVValue: getKVValueMapper,
  deleteKVValue: deleteKVValueMapper,
  deleteKVNamespace: deleteKVNamespaceMapper,
  createPool: createPoolMapper,
  updatePool: updatePoolMapper,
  getPool: getPoolMapper,
  deletePool: deletePoolMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onLoadBalancingHealthAlert: onLoadBalancingHealthAlertTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  createDnsRecord: buildActionStateRegistry("completed"),
  createMonitor: buildActionStateRegistry("completed"),
  createOriginRule: buildActionStateRegistry("created"),
  updateDNSRecord: buildActionStateRegistry("completed"),
  deleteDnsRecord: buildActionStateRegistry("completed"),
  deleteMonitor: buildActionStateRegistry("completed"),
  updateRedirectRule: buildActionStateRegistry("completed"),
  updateOriginRule: buildActionStateRegistry("updated"),
  deleteOriginRule: buildActionStateRegistry("deleted"),
  createKVNamespace: buildActionStateRegistry("created"),
  putKVValue: buildActionStateRegistry("success"),
  getKVValue: buildActionStateRegistry("fetched"),
  deleteKVValue: buildActionStateRegistry("deleted"),
  deleteKVNamespace: buildActionStateRegistry("deleted"),
  createPool: buildActionStateRegistry("created"),
  updatePool: buildActionStateRegistry("updated"),
  getPool: buildActionStateRegistry("fetched"),
  deletePool: buildActionStateRegistry("deleted"),
};
