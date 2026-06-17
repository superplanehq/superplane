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
import gcpIcon from "@/assets/icons/integrations/gcp.compute.svg";
import { renderTimeAgo } from "@/components/TimeAgo";
import { baseEventSections } from "./event_helpers";

interface DeleteLoadBalancerConfiguration {
  loadBalancer?: string;
}

interface DeleteLoadBalancerOutputData {
  forwardingRule?: string;
  backendService?: string;
  healthCheck?: string;
  region?: string;
  note?: string;
}

function lastSegment(value: string | undefined): string | undefined {
  if (!value) return undefined;
  const trimmed = value.trim();
  if (!trimmed || trimmed.includes("{{")) return undefined;
  const idx = trimmed.lastIndexOf("/");
  return idx >= 0 ? trimmed.slice(idx + 1).replace(/[?#].*$/, "") : trimmed;
}

function subtitle(context: SubtitleContext): string | React.ReactNode {
  const timestamp = context.execution.updatedAt || context.execution.createdAt;
  return timestamp ? renderTimeAgo(new Date(timestamp)) : "";
}

function baseProps(context: ComponentBaseContext, metadata: MetadataItem[]): ComponentBaseProps {
  const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
  const componentName = context.componentDefinition.name ?? "gcp";
  return {
    iconSrc: gcpIcon,
    iconSlug: context.componentDefinition?.icon ?? "trash-2",
    collapsedBackground: "bg-white",
    collapsed: context.node.isCollapsed,
    title: context.node.name || context.componentDefinition?.label || "Delete Load Balancer",
    eventSections: lastExecution ? baseEventSections(context.nodes, lastExecution, componentName) : undefined,
    metadata,
    includeEmptyState: !lastExecution,
    eventStateMap: getStateMap(componentName),
  };
}

function deleteMetadata(node: NodeInfo): MetadataItem[] {
  const config = node.configuration as DeleteLoadBalancerConfiguration | undefined;
  const label = lastSegment(config?.loadBalancer);
  return label ? [{ icon: "globe", label }] : [];
}

export const deleteLoadBalancerMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return baseProps(context, deleteMetadata(context.node));
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const details: Record<string, string> = {};
    if (context.execution.createdAt) {
      details["Executed At"] = new Date(context.execution.createdAt).toLocaleString();
    }
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const result = outputs?.default?.[0]?.data as DeleteLoadBalancerOutputData | undefined;
    if (!result) return details;

    if (result.forwardingRule) details["Forwarding Rule"] = result.forwardingRule;
    if (result.backendService) details["Backend Service"] = result.backendService;
    if (result.healthCheck) details["Health Check"] = result.healthCheck;
    if (result.region) details["Region"] = result.region;
    if (result.note) details["Note"] = result.note;
    return details;
  },

  subtitle,
};
