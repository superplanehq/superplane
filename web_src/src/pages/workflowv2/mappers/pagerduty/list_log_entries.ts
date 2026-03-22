import pdIcon from "@/assets/icons/integrations/pagerduty.svg";
import type React from "react";
import type { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import type { MetadataItem } from "@/ui/metadataList";
import { getBackgroundColorClass } from "@/utils/colors";
import { renderWithTimeAgo } from "@/components/TimeAgo";
import { getState, getStateMap, getTriggerRenderer } from "..";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  ExecutionInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import type { ListLogEntriesResponse, LogEntry } from "./types";

/**
 * Extracts the first payload from execution outputs.
 */
function getFirstPayload(execution: ExecutionInfo): OutputPayload | null {
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
function getLogEntries(execution: ExecutionInfo): LogEntry[] {
  const payload = getFirstPayload(execution);
  if (!payload || !payload.data) return [];

  const responseData = payload.data as ListLogEntriesResponse | undefined;
  if (!responseData || !responseData.log_entries) return [];

  return responseData.log_entries;
}

export const listLogEntriesMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution =
      context.lastExecutions && context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition?.name || "unknown";

    return {
      iconSrc: pdIcon,
      collapsedBackground: getBackgroundColorClass(context.componentDefinition?.color),
      collapsed: context.node?.isCollapsed ?? false,
      title:
        context.node?.name ||
        context.componentDefinition?.label ||
        context.componentDefinition?.name ||
        "Unnamed component",
      eventSections: lastExecution ? baseEventSections(context.nodes, lastExecution, componentName) : undefined,
      metadata: metadataList(context.node),
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    const date = new Date(context.execution.createdAt!);
    const logEntries = getLogEntries(context.execution);

    if (logEntries.length > 0) {
      return renderWithTimeAgo(`${logEntries.length} log entr${logEntries.length === 1 ? "y" : "ies"}`, date);
    }

    return renderWithTimeAgo("no log entries", date);
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, any> {
    const details: Record<string, any> = {};

    // Add "Checked at" timestamp
    if (context.execution.createdAt) {
      details["Checked at"] = new Date(context.execution.createdAt).toLocaleString();
    }

    const logEntries = getLogEntries(context.execution);
    details["Log Entries"] = `${logEntries.length} log entr${logEntries.length === 1 ? "y" : "ies"} fetched`;

    return details;
  },
};

function metadataList(node: { configuration?: unknown }): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  if (!node) return metadata;
  const configuration = node.configuration as any;

  if (configuration.incidentId) {
    metadata.push({ icon: "alert-triangle", label: `Incident: ${configuration.incidentId}` });
  }

  if (configuration.limit) {
    metadata.push({ icon: "hash", label: `Limit: ${configuration.limit}` });
  }

  return metadata;
}

function baseEventSections(nodes: { id: string }[], execution: ExecutionInfo, componentName: string): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer((rootTriggerNode as any)?.trigger?.name || "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent! });

  const logEntries = getLogEntries(execution);
  const date = new Date(execution.createdAt!);

  let eventSubtitle: string | React.ReactNode;
  if (logEntries.length > 0) {
    eventSubtitle = renderWithTimeAgo(`${logEntries.length} log entr${logEntries.length === 1 ? "y" : "ies"}`, date);
  } else {
    eventSubtitle = renderWithTimeAgo("no log entries", date);
  }

  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventSubtitle,
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent!.id,
    },
  ];
}
