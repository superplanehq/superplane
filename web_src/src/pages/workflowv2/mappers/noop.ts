import {
  ComponentsComponent,
  ComponentsNode,
  CanvasesCanvasNodeExecution,
  CanvasesCanvasNodeQueueItem,
} from "@/api-client";
import { ComponentBaseMapper, OutputPayload } from "./types";
import { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import { getTriggerRenderer, getStateMap } from ".";
import { formatTimeAgo } from "@/utils/date";
import { defaultStateFunction } from "./stateRegistry";

export const noopMapper: ComponentBaseMapper = {
  props(
    nodes: ComponentsNode[],
    node: ComponentsNode,
    componentDefinition: ComponentsComponent | undefined,
    lastExecutions: CanvasesCanvasNodeExecution[],
    _?: CanvasesCanvasNodeQueueItem[],
  ): ComponentBaseProps {
    const lastExecution = lastExecutions.length > 0 ? lastExecutions[0] : null;
    const componentName = componentDefinition?.name ?? "noop";

    return {
      iconSlug: componentDefinition?.icon ?? "circle-off",
      collapsed: node.isCollapsed,
      collapsedBackground: "bg-white",
      title: node.name || componentDefinition?.label || componentDefinition?.name || "Unnamed component",
      eventSections: lastExecution ? getNoopEventSections(nodes, lastExecution, componentName) : undefined,
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },
  subtitle(_node: ComponentsNode, execution: CanvasesCanvasNodeExecution): string {
    const timestamp = execution.updatedAt || execution.createdAt;
    return timestamp ? formatTimeAgo(new Date(timestamp)) : "";
  },
  getExecutionDetails(execution: CanvasesCanvasNodeExecution, _node: ComponentsNode): Record<string, string> {
    const details: Record<string, string> = {};
    const outputs = execution.outputs as { default?: OutputPayload[] } | undefined;
    const payload = outputs?.default?.[0];

    if (payload?.type) {
      details["Event Type"] = payload.type;
    }

    if (payload?.timestamp) {
      details["Emitted At"] = new Date(payload.timestamp).toLocaleString();
    }

    return details;
  },
};

function getNoopEventSections(
  nodes: ComponentsNode[],
  execution: CanvasesCanvasNodeExecution,
  _componentName: string,
): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.trigger?.name || "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle(execution.rootEvent!);
  const subtitleTimestamp = execution.updatedAt || execution.createdAt;
  const eventSubtitle = subtitleTimestamp ? formatTimeAgo(new Date(subtitleTimestamp)) : "";

  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventSubtitle,
      eventState: defaultStateFunction(execution),
      eventId: execution.rootEvent!.id!,
    },
  ];
}
