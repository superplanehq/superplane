import type {
  CanvasesCanvasNodeQueueItem,
  CanvasesCanvasRun,
  SuperplaneComponentsNode as ComponentsNode,
} from "@/api-client";
import { DEFAULT_EVENT_STATE_MAP } from "@/ui/componentBase";
import type { RunInspectorNodeSection } from "./types";

export function buildQueuedNodeSections({
  run,
  workflowNodes,
}: {
  run: CanvasesCanvasRun;
  workflowNodes: ComponentsNode[];
}): RunInspectorNodeSection[] {
  return [...(run.queueItems ?? [])]
    .filter((queueItem) => queueItem.id && queueItem.nodeId)
    .sort(compareQueueItemCreatedAt)
    .map((queueItem) => {
      const nodeId = queueItem.nodeId!;
      const workflowNode = workflowNodes.find((node) => node.id === nodeId);

      return {
        sectionValue: queuedSectionValue(queueItem),
        nodeId,
        nodeName: workflowNode?.name || nodeId,
        workflowNode,
        queueItem,
        isTrigger: false,
        isQueued: true,
        createdAt: queueItem.createdAt,
        badge: eventBadgeForQueuedItem(),
        tabData: null,
        upstreamSections: [],
        outputSections: [],
        actions: {
          canStop: false,
          canPushThrough: false,
          approvalRecords: [],
        },
        configurationFields: [],
      };
    });
}

function eventBadgeForQueuedItem(): { badgeColor: string; label: string } {
  const style = DEFAULT_EVENT_STATE_MAP.queued;
  return { badgeColor: style.badgeColor, label: style.label ?? "queued" };
}

function queuedSectionValue(queueItem: CanvasesCanvasNodeQueueItem): string {
  return `queue:${queueItem.nodeId}:${queueItem.id}`;
}

function compareQueueItemCreatedAt(left: CanvasesCanvasNodeQueueItem, right: CanvasesCanvasNodeQueueItem): number {
  return queueItemCreatedAt(left) - queueItemCreatedAt(right);
}

function queueItemCreatedAt(queueItem: CanvasesCanvasNodeQueueItem): number {
  const timestamp = queueItem.createdAt ? new Date(queueItem.createdAt).getTime() : Number.POSITIVE_INFINITY;
  return Number.isFinite(timestamp) ? timestamp : Number.POSITIVE_INFINITY;
}
