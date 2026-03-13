import { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import { getBackgroundColorClass } from "@/utils/colors";
import { getState, getStateMap, getTriggerRenderer } from "..";
import {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import { MetadataItem } from "@/ui/metadataList";
import doIcon from "@/assets/icons/integrations/digitalocean.svg";
import { formatTimeAgo } from "@/utils/date";
import { DropletNodeMetadata, GetDropletConfiguration } from "./types";

export const getDropletMapper: ComponentBaseMapper = {
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

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, any> {
    const details: Record<string, string> = {};

    if (context.execution.createdAt) {
      details["Executed At"] = new Date(context.execution.createdAt).toLocaleString();
    }

    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const droplet = outputs?.default?.[0]?.data as Record<string, any> | undefined;
    if (!droplet) return details;

    const ip = droplet.networks?.v4?.find((n: any) => n.type === "public")?.ip_address;

    details["Droplet ID"] = droplet.id?.toString() || "-";
    details["Name"] = droplet.name || "-";
    details["Status"] = droplet.status || "-";
    details["Region"] = droplet.region?.name || droplet.region?.slug || "-";
    details["Size"] = droplet.size_slug || "-";
    details["OS"] = droplet.image?.name || droplet.image?.slug || "-";
    details["Memory"] = droplet.memory ? `${droplet.memory} MB` : "-";
    details["vCPUs"] = droplet.vcpus?.toString() || "-";
    details["Disk"] = droplet.disk ? `${droplet.disk} GB` : "-";

    if (ip) {
      details["IP Address"] = ip;
    }

    if (droplet.tags && Array.isArray(droplet.tags) && droplet.tags.length > 0) {
      details["Tags"] = droplet.tags.join(", ");
    }

    return details;
  },

  subtitle(context: SubtitleContext): string {
    if (!context.execution.createdAt) return "";
    return formatTimeAgo(new Date(context.execution.createdAt));
  },
};

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const nodeMetadata = node.metadata as DropletNodeMetadata | undefined;
  const configuration = node.configuration as GetDropletConfiguration;

  if (nodeMetadata?.dropletName) {
    metadata.push({ icon: "hard-drive", label: nodeMetadata.dropletName });
  } else if (configuration?.dropletId) {
    metadata.push({ icon: "info", label: `Droplet ID: ${configuration.dropletId}` });
  }

  return metadata;
}

function baseEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName!);
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent! });

  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventSubtitle: formatTimeAgo(new Date(execution.createdAt!)),
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent!.id!,
    },
  ];
}
