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
import { DropletNodeMetadata } from "./types";

export const createSnapshotMapper: ComponentBaseMapper = {
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
    const snapshot = outputs?.default?.[0]?.data as Record<string, any> | undefined;
    if (!snapshot) return details;

    details["Snapshot ID"] = snapshot.id?.toString() || "-";
    details["Name"] = snapshot.name || "-";

    details["Resource ID"] = snapshot.resource_id?.toString() || "-";

    if (snapshot.regions?.length > 0) {
      details["Regions"] = snapshot.regions.join(", ");
    }

    if (snapshot.min_disk_size !== undefined) {
      details["Min Disk Size"] = `${snapshot.min_disk_size} GB`;
    }

    if (snapshot.size_gigabytes !== undefined) {
      details["Size"] = `${snapshot.size_gigabytes} GB`;
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
  const configuration = node.configuration as any;

  if (nodeMetadata?.dropletName) {
    metadata.push({ icon: "server", label: nodeMetadata.dropletName });
  } else if (configuration?.droplet) {
    metadata.push({ icon: "server", label: `Droplet ID: ${configuration.droplet}` });
  }

  if (configuration?.name) {
    metadata.push({ icon: "camera", label: `Snapshot: ${configuration.name}` });
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
