import { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import { getState, getStateMap, getTriggerRenderer } from "..";
import {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import dash0Icon from "@/assets/icons/integrations/dash0.svg";
import { MetadataItem } from "@/ui/metadataList";
import { formatTimeAgo } from "@/utils/date";
import { UpsertSyntheticCheckConfiguration } from "./types";

export const createSyntheticCheckMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name || "unknown";

    return {
      iconSrc: dash0Icon,
      collapsedBackground: "bg-white",
      collapsed: context.node.isCollapsed,
      title: context.node.name || context.componentDefinition.label || "Create Synthetic Check",
      eventSections: lastExecution ? baseEventSections(context.nodes, lastExecution, componentName) : undefined,
      metadata: metadataList(context.node),
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    return buildExecutionDetails(context.execution, "Create Response");
  },

  subtitle(context: SubtitleContext): string {
    const executionTime = context.execution.updatedAt ?? context.execution.createdAt;
    return formatTimeAgo(new Date(executionTime!));
  },
};

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as UpsertSyntheticCheckConfiguration;

  if (configuration?.originOrId) {
    metadata.push({
      icon: "hash",
      label: configuration.originOrId,
    });
  }

  return metadata;
}

function getFirstDefaultPayload(execution: ExecutionInfo): OutputPayload | null {
  const outputs = execution.outputs as { default?: OutputPayload[] } | undefined;
  if (!outputs?.default || outputs.default.length === 0) {
    return null;
  }

  return outputs.default[0] || null;
}

function buildExecutionDetails(execution: ExecutionInfo, label: string): Record<string, string> {
  const details: Record<string, string> = {};
  const payload = getFirstDefaultPayload(execution);

  if (payload?.timestamp) {
    details["Received At"] = new Date(payload.timestamp).toLocaleString();
  }

  if (!payload?.data) {
    details[label] = "No data returned";
    return details;
  }

  try {
    details[label] = JSON.stringify(payload.data, null, 2);
  } catch {
    details[label] = String(payload.data);
  }

  return details;
}

function baseEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  const rootTriggerNode = nodes.find((node) => node.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName!);
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });
  const executionTime = execution.updatedAt ?? execution.createdAt;

  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventSubtitle: formatTimeAgo(new Date(executionTime!)),
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent!.id!,
    },
  ];
}
