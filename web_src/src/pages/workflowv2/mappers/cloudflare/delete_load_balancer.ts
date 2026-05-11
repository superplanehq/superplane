import type { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import type React from "react";
import { getBackgroundColorClass } from "@/lib/colors";
import { getState, getStateMap, getTriggerRenderer } from "..";
import { deleteLoadBalancerExecutionDetails } from "./base";
import type { ComponentBaseContext, ComponentBaseMapper, ExecutionInfo, NodeInfo, SubtitleContext } from "../types";
import type { MetadataItem } from "@/ui/metadataList";
import cloudflareIcon from "@/assets/icons/integrations/cloudflare.svg";
import { renderTimeAgo } from "@/components/TimeAgo";

interface DeleteLoadBalancerConfiguration {
  loadBalancer?: string;
}

interface DeleteLoadBalancerNodeMetadata {
  loadBalancerName?: string;
}

export const deleteLoadBalancerMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name ?? "cloudflare";

    return {
      iconSrc: cloudflareIcon,
      collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
      collapsed: context.node.isCollapsed,
      title: context.node.name || context.componentDefinition.label || "Unnamed component",
      eventSections: lastExecution ? baseEventSections(context.nodes, lastExecution, componentName) : undefined,
      metadata: metadataList(context.node),
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails: deleteLoadBalancerExecutionDetails,

  subtitle(context: SubtitleContext): string | React.ReactNode {
    if (!context.execution.createdAt) return "";
    return renderTimeAgo(new Date(context.execution.createdAt));
  },
};

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const nodeMetadata = node.metadata as DeleteLoadBalancerNodeMetadata | undefined;
  const configuration = node.configuration as DeleteLoadBalancerConfiguration;

  const label = nodeMetadata?.loadBalancerName || configuration?.loadBalancer;
  if (label) {
    metadata.push({ icon: "network", label });
  }

  return metadata;
}

function baseEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  const receivedAt = execution.createdAt ? new Date(execution.createdAt) : new Date();
  const eventSubtitle = execution.createdAt ? renderTimeAgo(new Date(execution.createdAt)) : "";
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
