import {
  ComponentsNode,
  ComponentsComponent,
  CanvasesCanvasNodeExecution,
  CanvasesCanvasNodeQueueItem,
} from "@/api-client";
import { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import { getBackgroundColorClass } from "@/utils/colors";
import { getState, getStateMap, getTriggerRenderer } from "..";
import { ComponentBaseMapper, OutputPayload } from "../types";
import { MetadataItem } from "@/ui/metadataList";
import pdIcon from "@/assets/icons/integrations/pagerduty.svg";
import { formatTimeAgo } from "@/utils/date";
import { ListLogEntriesResponse, LogEntry } from "./types";

/**
 * Extracts the first payload from execution outputs.
 */
function getFirstPayload(execution: CanvasesCanvasNodeExecution): OutputPayload | null {
  const outputs = execution.outputs as { default?: OutputPayload[] } | undefined;
  if (!outputs) return null;

  if (outputs.default && outputs.default.length > 0) {
    return outputs.default[0];
  }

  return null;
}

/**
 * Extracts log entries from the execution payload.
 */
function getLogEntries(execution: CanvasesCanvasNodeExecution): LogEntry[] {
  const payload = getFirstPayload(execution);
  if (!payload || !payload.data) return [];

  const responseData = payload.data as ListLogEntriesResponse | undefined;
  if (!responseData || !responseData.log_entries) return [];

  return responseData.log_entries;
}

export const listLogEntriesMapper: ComponentBaseMapper = {
  props(
    nodes: ComponentsNode[],
    node: ComponentsNode,
    componentDefinition: ComponentsComponent,
    lastExecutions: CanvasesCanvasNodeExecution[],
    _?: CanvasesCanvasNodeQueueItem[],
  ): ComponentBaseProps {
    const lastExecution = lastExecutions.length > 0 ? lastExecutions[0] : null;
    const componentName = componentDefinition.name || node.component?.name || "unknown";

    return {
      iconSrc: pdIcon,
      collapsedBackground: getBackgroundColorClass(componentDefinition.color),
      collapsed: node.isCollapsed,
      title: node.name || componentDefinition.label || componentDefinition.name || "Unnamed component",
      eventSections: lastExecution ? baseEventSections(nodes, lastExecution, componentName) : undefined,
      metadata: metadataList(node),
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  subtitle(_node: ComponentsNode, execution: CanvasesCanvasNodeExecution): string {
    const timeAgo = formatTimeAgo(new Date(execution.createdAt!));
    const logEntries = getLogEntries(execution);

    if (logEntries.length > 0) {
      return `${logEntries.length} log entr${logEntries.length === 1 ? "y" : "ies"} · ${timeAgo}`;
    }

    return `no log entries · ${timeAgo}`;
  },

  getExecutionDetails(execution: CanvasesCanvasNodeExecution, _: ComponentsNode): Record<string, any> {
    const details: Record<string, any> = {};

    // Add "Checked at" timestamp
    if (execution.createdAt) {
      details["Checked at"] = new Date(execution.createdAt).toLocaleString();
    }

    const logEntries = getLogEntries(execution);
    details["Log Entries"] = `${logEntries.length} log entr${logEntries.length === 1 ? "y" : "ies"} fetched`;

    return details;
  },
};

function metadataList(node: ComponentsNode): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as any;

  if (configuration.incidentId) {
    metadata.push({ icon: "alert-triangle", label: `Incident: ${configuration.incidentId}` });
  }

  if (configuration.limit) {
    metadata.push({ icon: "hash", label: `Limit: ${configuration.limit}` });
  }

  return metadata;
}

function baseEventSections(
  nodes: ComponentsNode[],
  execution: CanvasesCanvasNodeExecution,
  componentName: string,
): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.trigger?.name || "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle(execution.rootEvent!);

  const logEntries = getLogEntries(execution);
  const timeAgo = formatTimeAgo(new Date(execution.createdAt!));

  let eventSubtitle: string;
  if (logEntries.length > 0) {
    eventSubtitle = `${logEntries.length} log entr${logEntries.length === 1 ? "y" : "ies"} · ${timeAgo}`;
  } else {
    eventSubtitle = `no log entries · ${timeAgo}`;
  }

  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventSubtitle,
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent!.id!,
    },
  ];
}
