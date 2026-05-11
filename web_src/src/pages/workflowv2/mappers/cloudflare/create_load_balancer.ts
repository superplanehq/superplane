import type { ComponentBaseProps } from "@/ui/componentBase";
import type React from "react";
import { getBackgroundColorClass } from "@/lib/colors";
import { getStateMap } from "..";
import { baseEventSections, getLoadBalancerExecutionDetails } from "./base";
import type { ComponentBaseContext, ComponentBaseMapper, NodeInfo, SubtitleContext } from "../types";
import type { MetadataItem } from "@/ui/metadataList";
import cloudflareIcon from "@/assets/icons/integrations/cloudflare.svg";
import { renderTimeAgo } from "@/components/TimeAgo";

interface CreateLoadBalancerConfiguration {
  name?: string;
  enabled?: boolean;
  defaultPools?: string[];
}

export const createLoadBalancerMapper: ComponentBaseMapper = {
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
  const configuration = node.configuration as CreateLoadBalancerConfiguration;

  if (configuration?.name) {
    metadata.push({ icon: "network", label: configuration.name });
  }

  if (configuration?.defaultPools != null && configuration.defaultPools.length > 0) {
    metadata.push({
      icon: "layers",
      label: `${configuration.defaultPools.length} pool${configuration.defaultPools.length === 1 ? "" : "s"}`,
    });
  }

  if (configuration?.enabled != null) {
    metadata.push({
      icon: configuration.enabled ? "check-circle" : "circle",
      label: configuration.enabled ? "Enabled" : "Disabled",
    });
  }

  return metadata;
}
