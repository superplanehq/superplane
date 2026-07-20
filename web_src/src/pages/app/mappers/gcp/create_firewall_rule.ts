import type { ComponentBaseProps } from "@/ui/componentBase";
import type React from "react";
import { getStateMap } from "..";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import type { MetadataItem } from "@/ui/metadataList";
import firewallIcon from "@/assets/icons/integrations/gcp.firewall.svg";
import { renderTimeAgo } from "@/components/TimeAgo";
import { baseEventSections } from "./event_helpers";
import { firewallLastSegment } from "./firewall_helpers";

interface CreateFirewallRuleConfiguration {
  name?: string;
  network?: string;
  direction?: string;
  action?: string;
}

interface CreateFirewallRuleOutputData {
  name?: string;
  network?: string;
  direction?: string;
  action?: string;
  priority?: number;
  link?: string;
}

function subtitle(context: SubtitleContext): string | React.ReactNode {
  const timestamp = context.execution.updatedAt || context.execution.createdAt;
  return timestamp ? renderTimeAgo(new Date(timestamp)) : "";
}

function baseProps(context: ComponentBaseContext, metadata: MetadataItem[]): ComponentBaseProps {
  const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
  const componentName = context.componentDefinition.name ?? "gcp";
  return {
    iconSrc: firewallIcon,
    iconSlug: context.componentDefinition?.icon ?? "shield",
    collapsedBackground: "bg-white",
    collapsed: context.node.isCollapsed,
    title: context.node.name || context.componentDefinition?.label || "Create Firewall Rule",
    eventSections: lastExecution ? baseEventSections(context.nodes, lastExecution, componentName) : undefined,
    metadata,
    includeEmptyState: !lastExecution,
    eventStateMap: getStateMap(componentName),
  };
}

function createMetadata(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const config = node.configuration as CreateFirewallRuleConfiguration | undefined;
  if (config?.name) metadata.push({ icon: "shield", label: config.name });
  const network = firewallLastSegment(config?.network);
  if (network) metadata.push({ icon: "network", label: network });
  if (config?.action) {
    const direction = config.direction ?? "INGRESS";
    metadata.push({ icon: "arrow-right-left", label: `${direction} · ${config.action.toUpperCase()}` });
  }
  return metadata;
}

export const createFirewallRuleMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return baseProps(context, createMetadata(context.node));
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const details: Record<string, string> = {};
    if (context.execution.createdAt) {
      details["Executed At"] = new Date(context.execution.createdAt).toLocaleString();
    }
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const result = outputs?.default?.[0]?.data as CreateFirewallRuleOutputData | undefined;
    if (!result) return details;

    if (result.name) details["Name"] = result.name;
    if (result.network) details["Network"] = result.network;
    if (result.direction) details["Direction"] = result.direction;
    if (result.action) details["Action"] = result.action;
    if (result.link) details["Console"] = result.link;
    return details;
  },

  subtitle,
};
