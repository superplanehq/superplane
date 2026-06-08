import type { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import { getBackgroundColorClass } from "@/lib/colors";
import { renderTimeAgo } from "@/components/TimeAgo";
import { getState, getStateMap, getTriggerRenderer } from "..";
import jiraIcon from "@/assets/icons/integrations/jira.svg";
import type { ComponentBaseContext, ExecutionInfo, NodeInfo } from "../types";
import type { MetadataItem } from "@/ui/metadataList";

export function jiraComponentBaseProps(context: ComponentBaseContext, metadata: MetadataItem[]): ComponentBaseProps {
  const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
  const componentName = context.componentDefinition.name || "jira";

  return {
    iconSrc: jiraIcon,
    collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
    collapsed: context.node.isCollapsed,
    title: context.node.name || context.componentDefinition.label || "Unnamed component",
    eventSections: lastExecution ? baseEventSections(context.nodes, lastExecution, componentName) : undefined,
    metadata,
    includeEmptyState: !lastExecution,
    eventStateMap: getStateMap(componentName),
  };
}

function buildJiraEventSections(
  nodes: NodeInfo[],
  execution: ExecutionInfo,
  componentName: string,
  fallbackWhenIncomplete: boolean,
): EventSection[] {
  const rootEvent = execution.rootEvent;
  const eventState = getState(componentName)(execution);

  if (!fallbackWhenIncomplete) {
    if (!rootEvent?.id || !execution.createdAt) return [];

    const rootTriggerNode = nodes.find((n) => n.id === rootEvent.nodeId);
    if (!rootTriggerNode?.componentName) return [];

    const { title } = getTriggerRenderer(rootTriggerNode.componentName).getTitleAndSubtitle({ event: rootEvent });
    return [
      {
        receivedAt: new Date(execution.createdAt),
        eventTitle: title,
        eventSubtitle: renderTimeAgo(new Date(execution.createdAt)),
        eventState,
        eventId: rootEvent.id,
      },
    ];
  }

  const receivedAt = execution.createdAt ? new Date(execution.createdAt) : new Date();
  const subtitleDate = execution.updatedAt ?? execution.createdAt;
  const eventSubtitle = subtitleDate ? renderTimeAgo(new Date(subtitleDate)) : "";

  const rootTriggerNode = nodes.find((n) => n.id === rootEvent?.nodeId);
  if (!rootTriggerNode || !rootEvent?.id) {
    return [{ receivedAt, eventTitle: "Execution", eventSubtitle, eventState, eventId: execution.id ?? "" }];
  }

  const { title } = getTriggerRenderer(rootTriggerNode.componentName).getTitleAndSubtitle({ event: rootEvent });
  return [{ receivedAt, eventTitle: title, eventSubtitle, eventState, eventId: rootEvent.id }];
}

export function baseEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  return buildJiraEventSections(nodes, execution, componentName, false);
}

export function jiraBaseEventSections(
  nodes: NodeInfo[],
  execution: ExecutionInfo,
  componentName: string,
): EventSection[] {
  return buildJiraEventSections(nodes, execution, componentName, true);
}

export function opsAlertCoreExecutionPayloadDetails(data: Record<string, unknown> | undefined): Record<string, string> {
  const out: Record<string, string> = {};
  if (!data) {
    return out;
  }
  if (data.message != null) out.Message = String(data.message);
  if (data.description != null) out.Description = String(data.description);
  if (data.status != null) out.Status = String(data.status);
  if (data.priority != null) out.Priority = String(data.priority);
  return out;
}

function trim(s: unknown): string {
  return typeof s === "string" ? s.trim() : "";
}

/** Card lines for Ops alert pickers — prefer enriched label from Setup when API fetch succeeded during save. */
export function buildOpsAlertReferenceMetadata(node: NodeInfo, alertRaw?: string): MetadataItem[] {
  const label = trim((node.metadata as { alertLabel?: string } | undefined)?.alertLabel);
  const id = trim(alertRaw ?? "");
  if (label !== "") return [{ icon: "hash", label }];
  if (id !== "") return [{ icon: "hash", label: id }];
  return [];
}
