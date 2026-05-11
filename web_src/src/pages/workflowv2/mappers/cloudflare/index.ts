import type { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { baseMapper } from "./base";
import { buildActionStateRegistry } from "../utils";
import { onLoadBalancingHealthAlertTriggerRenderer } from "./on_load_balancing_health_alert";
import { createMonitorMapper } from "./create_monitor";
import { deleteMonitorMapper } from "./delete_monitor";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  createDnsRecord: baseMapper,
  createMonitor: createMonitorMapper,
  updateDNSRecord: baseMapper,
  deleteDnsRecord: baseMapper,
  deleteMonitor: deleteMonitorMapper,
  updateRedirectRule: baseMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onLoadBalancingHealthAlert: onLoadBalancingHealthAlertTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  createDnsRecord: buildActionStateRegistry("completed"),
  createMonitor: buildActionStateRegistry("completed"),
  updateDNSRecord: buildActionStateRegistry("completed"),
  deleteDnsRecord: buildActionStateRegistry("completed"),
  deleteMonitor: buildActionStateRegistry("completed"),
  updateRedirectRule: buildActionStateRegistry("completed"),
};
