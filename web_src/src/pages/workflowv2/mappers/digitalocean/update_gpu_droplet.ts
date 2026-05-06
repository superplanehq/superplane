import type { ComponentBaseProps } from "@/ui/componentBase";
import type React from "react";
import { getBackgroundColorClass } from "@/lib/colors";
import { getStateMap } from "..";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  SubtitleContext,
} from "../types";
import type { MetadataItem } from "@/ui/metadataList";
import doIcon from "@/assets/icons/integrations/digitalocean.svg";
import { renderTimeAgo } from "@/components/TimeAgo";
import type { DropletNodeMetadata, UpdateGPUDropletConfiguration } from "./types";
import { buildBaseDropletDetails, findPublicIP, getDropletFromOutputs, gpuBaseEventSections } from "./gpu_droplet_base";

export const updateGPUDropletMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name ?? "digitalocean";

    return {
      iconSrc: doIcon,
      collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
      collapsed: context.node.isCollapsed,
      title: context.node.name || context.componentDefinition.label || "Unnamed component",
      eventSections: lastExecution ? baseEventSections(context.nodes, lastExecution, componentName) : undefined,
      metadata: metadataList(context.node),
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, unknown> {
    const details: Record<string, string> = {};

    if (context.execution.createdAt) {
      details["Executed At"] = new Date(context.execution.createdAt).toLocaleString();
    }

    const droplet = getDropletFromOutputs(context.execution.outputs);
    if (!droplet) return details;

    Object.assign(details, buildBaseDropletDetails(droplet));

    const ip = findPublicIP(droplet);
    if (ip) details["IP Address"] = ip;

    return details;
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    if (!context.execution.createdAt) return "";
    return renderTimeAgo(new Date(context.execution.createdAt));
  },
};

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const nodeMetadata = node.metadata as DropletNodeMetadata | undefined;
  const configuration = node.configuration as UpdateGPUDropletConfiguration;

  if (nodeMetadata?.dropletName) {
    metadata.push({ icon: "gpu", label: nodeMetadata.dropletName });
  } else if (configuration?.gpuDroplet) {
    metadata.push({ icon: "gpu", label: `GPU Droplet ID: ${configuration.gpuDroplet}` });
  }

  if (configuration?.name) {
    metadata.push({ icon: "edit", label: `Rename: ${configuration.name}` });
  }

  if (configuration?.gpuSize) {
    metadata.push({ icon: "arrow-up", label: `Upsize: ${configuration.gpuSize}` });
  }

  return metadata;
}

function baseEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string) {
  return gpuBaseEventSections(nodes, execution, componentName);
}
