import { renderTimeAgo } from "@/components/TimeAgo";
import type { EventSection } from "@/ui/componentBase";
import type { MetadataItem } from "@/ui/metadataList";
import { getState, getTriggerRenderer } from "..";
import type { DashboardNodeMetadata } from "./types";
import type { ExecutionInfo, NodeInfo } from "../types";

export function buildDashboardMetadata(node: NodeInfo): MetadataItem[] {
  const nodeMetadata = node.metadata as DashboardNodeMetadata | undefined;

  const items: (MetadataItem | undefined)[] = [
    nodeMetadata?.dashboardTitle ? { icon: "layout-dashboard", label: nodeMetadata.dashboardTitle } : undefined,
  ];

  return items.filter((item): item is MetadataItem => item !== undefined).slice(0, 3);
}

export function buildGrafanaEventSections(
  nodes: NodeInfo[],
  execution: ExecutionInfo,
  componentName: string,
): EventSection[] {
  const rootTriggerNode = nodes.find((node) => node.id === execution.rootEvent?.nodeId);
  const triggerName = rootTriggerNode?.componentName ?? "";
  const rootTriggerRenderer = getTriggerRenderer(triggerName);
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });

  const subtitleTimestamp = execution.updatedAt || execution.createdAt;
  const eventSubtitle = subtitleTimestamp ? renderTimeAgo(new Date(subtitleTimestamp)) : "";

  const receivedAtRaw = execution.createdAt || execution.updatedAt;
  const receivedAt = receivedAtRaw ? new Date(receivedAtRaw) : undefined;

  const eventId = execution.rootEvent?.id ?? execution.id;

  return [
    {
      receivedAt,
      eventTitle: title || "Trigger event",
      eventSubtitle,
      eventState: getState(componentName)(execution),
      eventId,
    },
  ];
}
