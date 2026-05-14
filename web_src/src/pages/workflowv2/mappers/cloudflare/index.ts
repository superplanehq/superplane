import { DEFAULT_EVENT_STATE_MAP } from "@/ui/componentBase";
import type { ComponentBaseMapper, EventStateRegistry, ExecutionInfo, OutputPayload, TriggerRenderer } from "../types";
import { defaultStateFunction } from "../stateRegistry";
import { baseMapper } from "./base";
import { buildActionStateRegistry } from "../utils";
import { onLoadBalancingHealthAlertTriggerRenderer } from "./on_load_balancing_health_alert";
import { createMonitorMapper } from "./create_monitor";
import { deleteMonitorMapper } from "./delete_monitor";
import { getMonitorMapper } from "./get_monitor";
import { updateMonitorMapper } from "./update_monitor";
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
import { orderCertificatePackMapper } from "./order_certificate_pack";
import { deleteCertificatePackMapper } from "./delete_certificate_pack";
import { createLoadBalancerMapper } from "./create_load_balancer";
import { getLoadBalancerMapper } from "./get_load_balancer";
import { updateLoadBalancerMapper } from "./update_load_balancer";
import { deleteLoadBalancerMapper } from "./delete_load_balancer";
import { deployWorkerMapper } from "./deploy_worker";
import { getWorkerMapper } from "./get_worker";
import { deleteWorkerMapper } from "./delete_worker";
import { updateWorkerRouteMapper } from "./update_worker_route";
import { createTunnelMapper } from "./create_tunnel";
import { getTunnelMapper } from "./get_tunnel";
import { deleteTunnelMapper } from "./delete_tunnel";
import { onTunnelHealthTriggerRenderer } from "./on_tunnel_health";

const updateWorkerRouteStateRegistry: EventStateRegistry = {
  stateMap: {
    ...DEFAULT_EVENT_STATE_MAP,
    created: DEFAULT_EVENT_STATE_MAP.success,
    updated: DEFAULT_EVENT_STATE_MAP.success,
  },
  getState: (execution: ExecutionInfo) => {
    const state = defaultStateFunction(execution);
    if (state !== "success") {
      return state;
    }
    const payloads = execution.outputs?.default as OutputPayload[] | undefined;
    const payloadType = payloads?.[0]?.type;
    if (payloadType === "cloudflare.workerRoute.updated") {
      return "updated";
    }
    return "created";
  },
};

const deployWorkerStateRegistry: EventStateRegistry = {
  stateMap: {
    ...DEFAULT_EVENT_STATE_MAP,
    deployed: DEFAULT_EVENT_STATE_MAP.success,
  },
  getState: (execution: ExecutionInfo) => {
    const state = defaultStateFunction(execution);
    if (state !== "success") {
      return state;
    }
    return "deployed";
  },
};

export const componentMappers: Record<string, ComponentBaseMapper> = {
  createDnsRecord: baseMapper,
  createMonitor: createMonitorMapper,
  getMonitor: getMonitorMapper,
  updateMonitor: updateMonitorMapper,
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
  deployWorker: deployWorkerMapper,
  getWorker: getWorkerMapper,
  deleteWorker: deleteWorkerMapper,
  updateWorkerRoute: updateWorkerRouteMapper,
  createTunnel: createTunnelMapper,
  getTunnel: getTunnelMapper,
  deleteTunnel: deleteTunnelMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onLoadBalancingHealthAlert: onLoadBalancingHealthAlertTriggerRenderer,
  onTunnelHealth: onTunnelHealthTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  createDnsRecord: buildActionStateRegistry("completed"),
  createMonitor: buildActionStateRegistry("created"),
  getMonitor: buildActionStateRegistry("fetched"),
  updateMonitor: buildActionStateRegistry("updated"),
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
  deployWorker: deployWorkerStateRegistry,
  getWorker: buildActionStateRegistry("fetched"),
  deleteWorker: buildActionStateRegistry("deleted"),
  updateWorkerRoute: updateWorkerRouteStateRegistry,
  createTunnel: buildActionStateRegistry("created"),
  getTunnel: buildActionStateRegistry("fetched"),
  deleteTunnel: buildActionStateRegistry("deleted"),
};
