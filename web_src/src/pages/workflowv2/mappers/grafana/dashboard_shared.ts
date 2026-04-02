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
  if (!execution.rootEvent?.id || !execution.createdAt) {
    return [];
  }

  const rootTriggerNode = nodes.find((node) => node.id === execution.rootEvent?.nodeId);
  if (!rootTriggerNode?.componentName) {
    return [];
  }

  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode.componentName);
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });

  return [
    {
      receivedAt: new Date(execution.createdAt),
      eventTitle: title || "Trigger event",
      eventSubtitle: renderTimeAgo(new Date(execution.createdAt)),
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent.id,
    },
  ];
}
