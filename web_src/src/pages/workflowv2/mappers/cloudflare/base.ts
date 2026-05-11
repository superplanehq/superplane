import type { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import type React from "react";
import { getState, getStateMap, getTriggerRenderer } from "..";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import cloudflareIcon from "@/assets/icons/integrations/cloudflare.svg";
import { renderTimeAgo } from "@/components/TimeAgo";

export const baseMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name ?? "cloudflare";

    return {
      iconSrc: cloudflareIcon,
      iconSlug: context.componentDefinition?.icon ?? "cloud",
      collapsedBackground: "bg-white",
      collapsed: context.node.isCollapsed,
      title: context.node.name || context.componentDefinition?.label || "Cloudflare",
      eventSections: lastExecution ? baseEventSections(context.nodes, lastExecution, componentName) : undefined,
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const details: Record<string, string> = {};
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const payload = outputs?.default?.[0];

    if (payload?.type) {
      details["Event Type"] = payload.type;
    }

    if (payload?.timestamp) {
      details["Emitted At"] = new Date(payload.timestamp).toLocaleString();
    }

    return details;
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    const timestamp = context.execution.updatedAt || context.execution.createdAt;
    return timestamp ? renderTimeAgo(new Date(timestamp)) : "";
  },
};

export function getPoolExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
  const details: Record<string, string> = {};

  if (context.execution.createdAt) {
    details["Executed At"] = new Date(context.execution.createdAt).toLocaleString();
  }

  const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
  const result = outputs?.default?.[0]?.data as Record<string, unknown> | undefined;
  const pool = result?.pool as Record<string, unknown> | undefined;
  if (!pool) return details;

  details["Pool ID"] = pool.id != null ? String(pool.id) : "-";
  details["Name"] = pool.name != null ? String(pool.name) : "-";

  if (pool.description != null) {
    details["Description"] = String(pool.description);
  }

  details["Enabled"] = pool.enabled != null ? String(pool.enabled) : "-";
  details["Minimum Origins"] = pool.minimum_origins != null ? String(pool.minimum_origins) : "-";
  details["Number of Origins"] = Array.isArray(pool.origins) ? String(pool.origins.length) : "-";

  return details;
}

export function getLoadBalancerExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
  const details: Record<string, string> = {};

  if (context.execution.createdAt) {
    details["Executed At"] = new Date(context.execution.createdAt).toLocaleString();
  }

  const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
  const result = outputs?.default?.[0]?.data as Record<string, unknown> | undefined;
  const lb = result?.loadBalancer as Record<string, unknown> | undefined;
  if (!lb) return details;

  details["Name"] = lb.name != null ? String(lb.name) : "-";

  if (lb.description != null) {
    details["Description"] = String(lb.description);
  }

  details["Enabled"] = lb.enabled != null ? String(lb.enabled) : "-";
  details["Proxied"] = lb.proxied != null ? String(lb.proxied) : "-";
  details["Steering Policy"] = lb.steering_policy != null ? String(lb.steering_policy) : "-";
  details["Session Affinity"] = lb.session_affinity != null ? String(lb.session_affinity) : "-";
  details["Default Pools"] = Array.isArray(lb.default_pools) ? String(lb.default_pools.length) : "-";

  return details;
}

export function updateLoadBalancerExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
  const details: Record<string, string> = {};

  if (context.execution.createdAt) {
    details["Executed At"] = new Date(context.execution.createdAt).toLocaleString();
  }

  const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
  const result = outputs?.default?.[0]?.data as Record<string, unknown> | undefined;
  const lb = result?.loadBalancer as Record<string, unknown> | undefined;
  if (!lb) return details;

  details["Name"] = lb.name != null ? String(lb.name) : "-";

  if (lb.description != null) {
    details["Description"] = String(lb.description);
  }

  details["Enabled"] = lb.enabled != null ? String(lb.enabled) : "-";
  details["Steering Policy"] = lb.steering_policy != null ? String(lb.steering_policy) : "-";
  details["Session Affinity"] = lb.session_affinity != null ? String(lb.session_affinity) : "-";
  details["Default Pools"] = Array.isArray(lb.default_pools) ? String(lb.default_pools.length) : "-";

  return details;
}

export function deleteLoadBalancerExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
  const details: Record<string, string> = {};

  if (context.execution.createdAt) {
    details["Executed At"] = new Date(context.execution.createdAt).toLocaleString();
  }

  const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
  const result = outputs?.default?.[0]?.data as Record<string, unknown> | undefined;
  if (!result) return details;

  details["Deleted"] = result.deleted != null ? String(result.deleted) : "-";

  return details;
}

export function baseEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  const receivedAt = execution.createdAt ? new Date(execution.createdAt) : new Date();
  const subtitleDate = execution.updatedAt ?? execution.createdAt;
  const eventSubtitle = subtitleDate ? renderTimeAgo(new Date(subtitleDate)) : "";
  const eventState = getState(componentName)(execution);

  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  if (!rootTriggerNode || !execution.rootEvent?.id) {
    return [{ receivedAt, eventTitle: "Execution", eventSubtitle, eventState, eventId: execution.id ?? "" }];
  }

  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode.componentName);
  if (!rootTriggerRenderer) {
    return [{ receivedAt, eventTitle: "Execution", eventSubtitle, eventState, eventId: execution.rootEvent.id }];
  }

  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });
  return [{ receivedAt, eventTitle: title, eventSubtitle, eventState, eventId: execution.rootEvent.id }];
}
