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
import type { CreateGPUDropletConfiguration } from "./types";
import { buildBaseDropletDetails, findPublicIP, getDropletFromOutputs, gpuBaseEventSections } from "./gpu_droplet_base";

export const createGPUDropletMapper: ComponentBaseMapper = {
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
      details["Created At"] = new Date(context.execution.createdAt).toLocaleString();
    }

    const droplet = getDropletFromOutputs(context.execution.outputs);
    if (!droplet) return details;

    const base = buildBaseDropletDetails(droplet);
    details["Droplet ID"] = base["Droplet ID"];
    details["Name"] = base["Name"];
    details["Region"] = base["Region"];
    details["GPU Size"] = base["GPU Size"];
    details["OS"] = droplet.image?.name || droplet.image?.slug || "-";

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
  const configuration = node.configuration as CreateGPUDropletConfiguration;

  if (configuration?.gpuRegion) {
    metadata.push({ icon: "map-pin", label: `Region: ${configuration.gpuRegion}` });
  }

  if (configuration?.gpuSize) {
    metadata.push({ icon: "gpu", label: `GPU Size: ${configuration.gpuSize}` });
  }

  const image = configuration?.oneClickGPUImage || configuration?.baseGPUImage;
  if (image) {
    metadata.push({ icon: "hard-drive", label: `Image: ${image}` });
  }

  return metadata;
}

function baseEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string) {
  return gpuBaseEventSections(nodes, execution, componentName);
}
