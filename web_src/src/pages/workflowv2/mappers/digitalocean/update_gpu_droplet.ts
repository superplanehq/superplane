import type { ComponentBaseProps } from "@/ui/componentBase";
import type React from "react";
import { getBackgroundColorClass } from "@/utils/colors";
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
import doIcon from "@/assets/icons/integrations/digitalocean.svg";
import { renderTimeAgo } from "@/components/TimeAgo";
import { baseEventSections } from "./utils";
import type { DropletNodeMetadata, UpdateGPUDropletConfiguration, DropletData } from "./types";

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

    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const droplet = outputs?.default?.[0]?.data as DropletData | undefined;
    if (!droplet) return details;

    const ip = droplet.networks?.v4?.find((n) => n.type === "public")?.ip_address;

    details["Droplet ID"] = droplet.id?.toString() || "-";
    details["Name"] = droplet.name || "-";
    details["Status"] = droplet.status || "-";
    details["Region"] = droplet.region?.name || droplet.region?.slug || "-";
    details["GPU Size"] = droplet.size_slug || "-";
    details["Memory"] = droplet.memory ? `${droplet.memory} MB` : "-";
    details["vCPUs"] = droplet.vcpus?.toString() || "-";
    details["Disk"] = droplet.disk ? `${droplet.disk} GB` : "-";

    if (ip) {
      details["IP Address"] = ip;
    }

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
  } else if (configuration?.droplet) {
    metadata.push({ icon: "gpu", label: `GPU Droplet ID: ${configuration.droplet}` });
  }

  if (configuration?.name) {
    metadata.push({ icon: "edit", label: `Rename: ${configuration.name}` });
  }

  if (configuration?.size) {
    metadata.push({ icon: "arrow-up", label: `Upsize: ${configuration.size}` });
  }

  return metadata;
}
