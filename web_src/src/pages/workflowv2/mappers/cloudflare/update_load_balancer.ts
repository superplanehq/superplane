import type { ComponentBaseProps } from "@/ui/componentBase";
import type React from "react";
import { getBackgroundColorClass } from "@/lib/colors";
import { getStateMap } from "..";
import { baseEventSections, getLoadBalancerExecutionDetails } from "./base";
import type { ComponentBaseContext, ComponentBaseMapper, NodeInfo, SubtitleContext } from "../types";
import type { MetadataItem } from "@/ui/metadataList";
import cloudflareIcon from "@/assets/icons/integrations/cloudflare.svg";
import { renderTimeAgo } from "@/components/TimeAgo";

interface UpdateLoadBalancerConfiguration {
  loadBalancer?: string;
  description?: string;
  steeringPolicy?: string;
  defaultPools?: string[];
  enabled?: boolean;
}

interface UpdateLoadBalancerNodeMetadata {
  loadBalancerName?: string;
}

export const updateLoadBalancerMapper: ComponentBaseMapper = {
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

  getExecutionDetails: getLoadBalancerExecutionDetails,

  subtitle(context: SubtitleContext): string | React.ReactNode {
    if (!context.execution.createdAt) return "";
    return renderTimeAgo(new Date(context.execution.createdAt));
  },
};

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const nodeMetadata = node.metadata as UpdateLoadBalancerNodeMetadata | undefined;
  const config = node.configuration as UpdateLoadBalancerConfiguration | undefined;

  const label = nodeMetadata?.loadBalancerName || config?.loadBalancer;
  if (label) {
    metadata.push({ icon: "network", label });
  }

  if (config?.description) {
    metadata.push({ icon: "text", label: config.description });
  }

  if (config?.steeringPolicy) {
    metadata.push({ icon: "git-branch", label: config.steeringPolicy });
  }

  if (config?.defaultPools != null && config.defaultPools.length > 0) {
    const count = config.defaultPools.length;
    metadata.push({ icon: "layers", label: `${count} pool${count === 1 ? "" : "s"}` });
  }

  if (config?.enabled != null) {
    metadata.push({
      icon: config.enabled ? "check-circle" : "circle",
      label: config.enabled ? "Enabled" : "Disabled",
    });
  }

  return metadata;
}
