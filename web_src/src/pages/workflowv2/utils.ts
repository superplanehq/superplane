import { WorkflowsWorkflowEvent, WorkflowsWorkflowNodeExecution, ComponentsNode } from "@/api-client";
import { SidebarEvent } from "@/ui/CanvasPage";
import { getTriggerRenderer } from "./renderers";
import { formatTimeAgo } from "@/utils/date";

export function mapTriggerEventsToSidebarEvents(events: WorkflowsWorkflowEvent[], node: ComponentsNode, limit?: number): SidebarEvent[] {
  const eventsToMap = limit ? events.slice(0, limit) : events;
  return eventsToMap.map((event) => {
    const triggerRenderer = getTriggerRenderer(node.trigger?.name || "");
    const { title, subtitle } = triggerRenderer.getTitleAndSubtitle(event);
    const values = triggerRenderer.getRootEventValues(event);

    return {
      id: event.id!,
      title,
      subtitle,
      state: "processed" as const,
      isOpen: false,
      receivedAt: event.createdAt ? new Date(event.createdAt) : undefined,
      values,
    };
  });
}

export function mapExecutionsToSidebarEvents(executions: WorkflowsWorkflowNodeExecution[], nodes: ComponentsNode[], limit?: number): SidebarEvent[] {
  const executionsToMap = limit ? executions.slice(0, limit) : executions;
  return executionsToMap.map((execution) => {
    const state =
      execution.state === "STATE_FINISHED" && execution.result === "RESULT_PASSED"
        ? ("processed" as const)
        : execution.state === "STATE_FINISHED" && execution.result === "RESULT_FAILED"
          ? ("discarded" as const)
          : ("waiting" as const);

    const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
    const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.trigger?.name || "");

    const { title, subtitle } = execution.rootEvent
      ? rootTriggerRenderer.getTitleAndSubtitle(execution.rootEvent)
      : {
          title: execution.id || "Execution",
          subtitle: execution.createdAt ? formatTimeAgo(new Date(execution.createdAt)).replace(" ago", "") : "",
        };

    const values = execution.rootEvent ? rootTriggerRenderer.getRootEventValues(execution.rootEvent) : {};

    return {
      id: execution.id!,
      title,
      subtitle,
      state,
      isOpen: false,
      receivedAt: execution.createdAt ? new Date(execution.createdAt) : undefined,
      values,
    };
  });
}