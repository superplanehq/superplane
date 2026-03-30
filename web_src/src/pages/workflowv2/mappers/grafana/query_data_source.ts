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
import type { QueryDataSourceConfiguration } from "./types";
import { renderTimeAgo } from "@/components/TimeAgo";
import { formatOptionalIsoTimestamp } from "@/lib/timezone";

export const queryDataSourceMapper: ComponentBaseMapper = {
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
    const configuration = context.node.configuration as QueryDataSourceConfiguration | undefined;
    const details: Record<string, string> = {
      "Queried At": formatOptionalIsoTimestamp(context.execution.createdAt),
    };

    if (configuration?.dataSourceUid) {
      details["Data Source"] = configuration.dataSourceUid;
    }

    if (configuration?.query) {
      details.Query = configuration.query;
    }

    if (configuration?.format) {
      details.Format = configuration.format;
    }

    if (!outputs || !outputs.default || outputs.default.length === 0) {
      details.Response = "No data returned";
      return details;
    }

    const payload = outputs.default[0];
    const responseData = payload?.data as Record<string, any> | undefined;
    const payloadTimestamp = formatOptionalIsoTimestamp(payload?.timestamp);
    if (payloadTimestamp !== "-") {
      details["Queried At"] = payloadTimestamp;
    }

    if (!responseData) {
      details.Response = "No data returned";
      return details;
    }

    return {
      ...details,
      ...buildQueryResultSummary(responseData),
    };
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    if (!context.execution.createdAt) return "-";
    return renderTimeAgo(new Date(context.execution.createdAt));
  },
};

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as QueryDataSourceConfiguration;

  if (configuration?.dataSourceUid) {
    metadata.push({ icon: "database", label: `Data Source: ${configuration.dataSourceUid}` });
  }

  if (configuration?.query) {
    const preview =
      configuration.query.length > 50 ? configuration.query.substring(0, 50) + "..." : configuration.query;
    metadata.push({ icon: "code", label: preview });
  }

  if (configuration?.format) {
    metadata.push({ icon: "funnel", label: `Format: ${configuration.format}` });
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

function buildQueryResultSummary(responseData: Record<string, any>): Record<string, string> {
  const details: Record<string, string> = {};
  const results = responseData.results;

  if (!results || typeof results !== "object" || Array.isArray(results)) {
    details["Response Keys"] = Object.keys(responseData).join(", ") || "-";
    return details;
  }

  const refIds = Object.keys(results);
  if (refIds.length === 0) {
    details.Results = "No results";
    return details;
  }

  details["Result Ref IDs"] = refIds.join(", ");
  details.Results = String(refIds.length);

  let frameCount = 0;
  let rowCount = 0;
  const fieldNames = new Set<string>();

  for (const refId of refIds) {
    const result = results[refId];
    if (!result || typeof result !== "object" || Array.isArray(result)) {
      continue;
    }

    const frames = Array.isArray(result.frames) ? result.frames : [];
    frameCount += frames.length;

    for (const frame of frames) {
      if (!frame || typeof frame !== "object" || Array.isArray(frame)) {
        continue;
      }

      const nextRowCount = getFrameRowCount(frame);
      rowCount += nextRowCount;

      const schemaFields = Array.isArray(frame.schema?.fields) ? frame.schema.fields : [];
      for (const field of schemaFields) {
        if (field?.name && typeof field.name === "string") {
          fieldNames.add(field.name);
        }
      }
    }
  }

  details["Frame Count"] = String(frameCount);

  if (rowCount > 0) {
    details["Row Count"] = String(rowCount);
  }

  if (fieldNames.size > 0) {
    details.Fields = Array.from(fieldNames).slice(0, 5).join(", ");
  }

  return details;
}

function getFrameRowCount(frame: Record<string, any>): number {
  const values = frame.data?.values;
  if (!Array.isArray(values)) {
    return 0;
  }

  let maxLength = 0;
  for (const column of values) {
    if (Array.isArray(column) && column.length > maxLength) {
      maxLength = column.length;
    }
  }

  return maxLength;
}

