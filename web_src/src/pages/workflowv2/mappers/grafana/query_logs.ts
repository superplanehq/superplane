import type { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import type React from "react";
import { getState, getStateMap, getTriggerRenderer } from "..";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import type { MetadataItem } from "@/ui/metadataList";
import grafanaIcon from "@/assets/icons/integrations/grafana.svg";
import type { QueryLogsConfiguration } from "./types";
import { renderTimeAgo } from "@/components/TimeAgo";
import { formatTimestamp } from "../utils";

export const queryLogsMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name || "unknown";

    return {
      iconSrc: grafanaIcon,
      collapsedBackground: "bg-white",
      collapsed: context.node.isCollapsed,
      title: context.node.name || context.componentDefinition.label || "Unnamed component",
      eventSections: lastExecution ? baseEventSections(context.nodes, lastExecution, componentName) : undefined,
      metadata: metadataList(context.node),
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const configuration = context.node.configuration as QueryLogsConfiguration | undefined;
    const details: Record<string, string> = {
      "Queried At": formatTimestamp(context.execution.createdAt),
    };

    if (configuration?.dataSourceUid) {
      details["Data Source"] = configuration.dataSourceUid;
    }

    if (configuration?.query) {
      details["Query"] =
        configuration.query.length > 80 ? configuration.query.substring(0, 80) + "..." : configuration.query;
    }

    if (!outputs || !outputs.default || outputs.default.length === 0) {
      details["Log Lines"] = "0";
      return details;
    }

    const payload = outputs.default[0];
    const payloadTimestamp = formatTimestamp(payload?.timestamp);
    if (payloadTimestamp !== "-") {
      details["Queried At"] = payloadTimestamp;
    }

    const responseData = payload?.data as Record<string, unknown> | undefined;
    if (responseData) {
      const lineCount = countLogLines(responseData);
      details["Log Lines"] = String(lineCount);
    }

    return details;
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    if (!context.execution.createdAt) return "-";
    return renderTimeAgo(new Date(context.execution.createdAt));
  },
};

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as QueryLogsConfiguration | undefined;

  if (configuration?.dataSourceUid) {
    metadata.push({ icon: "database", label: `Data Source: ${configuration.dataSourceUid}` });
  }

  if (configuration?.query) {
    const preview =
      configuration.query.length > 50 ? configuration.query.substring(0, 50) + "..." : configuration.query;
    metadata.push({ icon: "code", label: preview });
  }

  return metadata;
}

function baseEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName || "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });
  const eventTitle = title || "Trigger event";

  return [
    {
      receivedAt: execution.createdAt ? new Date(execution.createdAt) : undefined,
      eventTitle: eventTitle,
      eventSubtitle: execution.createdAt ? renderTimeAgo(new Date(execution.createdAt)) : "-",
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent?.id || "",
    },
  ];
}

function countLogLines(responseData: Record<string, unknown>): number {
  const results = responseData.results;
  if (!results || typeof results !== "object" || Array.isArray(results)) {
    return 0;
  }

  let total = 0;
  for (const refId of Object.keys(results)) {
    const result = (results as Record<string, unknown>)[refId];
    if (!result || typeof result !== "object" || Array.isArray(result)) continue;

    const frames = (result as Record<string, unknown>).frames;
    if (!Array.isArray(frames)) continue;

    for (const frame of frames) {
      if (!frame || typeof frame !== "object" || Array.isArray(frame)) continue;
      const data = (frame as Record<string, unknown>).data;
      if (!data || typeof data !== "object" || Array.isArray(data)) continue;
      const values = (data as Record<string, unknown>).values;
      if (!Array.isArray(values) || values.length === 0) continue;
      const firstCol = values[0];
      if (Array.isArray(firstCol)) {
        total += firstCol.length;
      }
    }
  }

  return total;
}
