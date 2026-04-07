import type { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import React from "react";
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
import type { ListSilencesOutput } from "./types";
import { renderTimeAgo } from "@/components/TimeAgo";
import { formatTimestamp } from "../utils";

export const listSilencesMapper: ComponentBaseMapper = {
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
    const details: Record<string, string> = {
      "Listed At": formatTimestamp(context.execution.createdAt),
    };

    if (!outputs || !outputs.default || outputs.default.length === 0) {
      details["Silence Count"] = "0";
      return details;
    }

    const payload = outputs.default[0];
    const payloadTimestamp = formatTimestamp(payload?.timestamp);
    if (payloadTimestamp !== "-") {
      details["Listed At"] = payloadTimestamp;
    }

    const output = payload?.data as ListSilencesOutput | undefined;
    const silences = output?.silences ?? [];
    const count = silences.length;

    details["Silence Count"] = String(count);

    if (count > 0 && silences[0].comment) {
      details["First Comment"] = silences[0].comment;
    }

    return details;
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const payload = outputs?.default?.[0];
    const output = payload?.data as ListSilencesOutput | undefined;
    const count = output?.silences?.length ?? 0;

    const timeAgo = context.execution.createdAt ? renderTimeAgo(new Date(context.execution.createdAt)) : "-";

    return (
      <span>
        {count} silence{count === 1 ? "" : "s"} · {timeAgo}
      </span>
    );
  },
};

function metadataList(_node: NodeInfo): MetadataItem[] {
  return [];
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
