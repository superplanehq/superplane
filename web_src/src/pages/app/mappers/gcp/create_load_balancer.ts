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

interface CreateLoadBalancerConfiguration {
  name?: string;
  region?: string;
  protocol?: string;
  ports?: string[];
}

interface CreateLoadBalancerOutputData {
  name?: string;
  region?: string;
  ipAddress?: string;
  protocol?: string;
  ports?: string[];
  forwardingRule?: string;
  backendService?: string;
  healthCheck?: string;
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
    iconSlug: context.componentDefinition?.icon ?? "globe",
    collapsedBackground: "bg-white",
    collapsed: context.node.isCollapsed,
    title: context.node.name || context.componentDefinition?.label || "Create Load Balancer",
    eventSections: lastExecution ? baseEventSections(context.nodes, lastExecution, componentName) : undefined,
    metadata,
    includeEmptyState: !lastExecution,
    eventStateMap: getStateMap(componentName),
  };
}

function createMetadata(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const config = node.configuration as CreateLoadBalancerConfiguration | undefined;
  if (config?.name) metadata.push({ icon: "globe", label: config.name });
  if (config?.region) metadata.push({ icon: "map-pin", label: config.region });
  const ports = config?.ports?.length ? config.ports.join(", ") : undefined;
  if (ports) metadata.push({ icon: "ethernet-port", label: `${config?.protocol ?? "TCP"} ${ports}` });
  return metadata;
}

export const createLoadBalancerMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return baseProps(context, createMetadata(context.node));
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const details: Record<string, string> = {};
    if (context.execution.createdAt) {
      details["Executed At"] = new Date(context.execution.createdAt).toLocaleString();
    }
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const result = outputs?.default?.[0]?.data as CreateLoadBalancerOutputData | undefined;
    if (!result) return details;

    if (result.name) details["Name"] = result.name;
    if (result.region) details["Region"] = result.region;
    if (result.ipAddress) details["IP Address"] = result.ipAddress;
    if (result.backendService) details["Backend Service"] = result.backendService;
    return details;
  },

  subtitle,
};
