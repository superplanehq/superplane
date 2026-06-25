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
import { firewallLastSegment, type FirewallNodeMetadata } from "./firewall_helpers";

interface UpdateFirewallRuleConfiguration {
  firewall?: string;
  enabledState?: string;
}

interface UpdateFirewallRuleOutputData {
  name?: string;
  network?: string;
  priority?: number;
  disabled?: boolean;
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
    title: context.node.name || context.componentDefinition?.label || "Update Firewall Rule",
    eventSections: lastExecution ? baseEventSections(context.nodes, lastExecution, componentName) : undefined,
    metadata,
    includeEmptyState: !lastExecution,
    eventStateMap: getStateMap(componentName),
  };
}

function updateMetadata(node: NodeInfo): MetadataItem[] {
  const nodeMetadata = node.metadata as FirewallNodeMetadata | undefined;
  const config = node.configuration as UpdateFirewallRuleConfiguration | undefined;
  const label = nodeMetadata?.firewallName || firewallLastSegment(config?.firewall);
  return label ? [{ icon: "shield", label }] : [];
}

export const updateFirewallRuleMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return baseProps(context, updateMetadata(context.node));
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const details: Record<string, string> = {};
    if (context.execution.createdAt) {
      details["Executed At"] = new Date(context.execution.createdAt).toLocaleString();
    }
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const result = outputs?.default?.[0]?.data as UpdateFirewallRuleOutputData | undefined;
    if (!result) return details;

    if (result.name) details["Name"] = result.name;
    if (result.network) details["Network"] = result.network;
    if (result.priority !== undefined) details["Priority"] = String(result.priority);
    if (result.disabled !== undefined) details["Enabled"] = result.disabled ? "No" : "Yes";
    if (result.link) details["Console"] = result.link;
    return details;
  },

  subtitle,
};
