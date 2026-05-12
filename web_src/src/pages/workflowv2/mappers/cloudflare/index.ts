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
import { purgeCacheMapper } from "./purge_cache";
import { orderCertificatePackMapper, deleteCertificatePackMapper } from "./certificate_pack";
import { createLoadBalancerMapper } from "./create_load_balancer";
import { getLoadBalancerMapper } from "./get_load_balancer";
import { updateLoadBalancerMapper } from "./update_load_balancer";
import { deleteLoadBalancerMapper } from "./delete_load_balancer";

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
  purgeCache: purgeCacheMapper,
  orderCertificatePack: orderCertificatePackMapper,
  deleteCertificatePack: deleteCertificatePackMapper,
  createLoadBalancer: createLoadBalancerMapper,
  getLoadBalancer: getLoadBalancerMapper,
  updateLoadBalancer: updateLoadBalancerMapper,
  deleteLoadBalancer: deleteLoadBalancerMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onLoadBalancingHealthAlert: onLoadBalancingHealthAlertTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  createDnsRecord: buildActionStateRegistry("completed"),
  createMonitor: buildActionStateRegistry("created"),
  createOriginRule: buildActionStateRegistry("created"),
  updateDNSRecord: buildActionStateRegistry("completed"),
  deleteDnsRecord: buildActionStateRegistry("completed"),
  deleteMonitor: buildActionStateRegistry("deleted"),
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
  purgeCache: buildActionStateRegistry("purged"),
  orderCertificatePack: buildActionStateRegistry("ordered"),
  deleteCertificatePack: buildActionStateRegistry("deleted"),
  createLoadBalancer: buildActionStateRegistry("created"),
  getLoadBalancer: buildActionStateRegistry("fetched"),
  updateLoadBalancer: buildActionStateRegistry("updated"),
  deleteLoadBalancer: buildActionStateRegistry("deleted"),
};
