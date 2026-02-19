import { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import { getState, getStateMap, getTriggerRenderer } from "..";
import {
  ComponentBaseMapper,
  ExecutionDetailsContext,
  ComponentBaseContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import { MetadataItem } from "@/ui/metadataList";
import dash0Icon from "@/assets/icons/integrations/dash0.svg";
import { DeleteHttpSyntheticCheckConfiguration } from "./types";
import { formatTimeAgo } from "@/utils/date";

export const deleteHttpSyntheticCheckMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name || "unknown";

    return {
      iconSrc: dash0Icon,
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

    if (!outputs || !outputs.default || outputs.default.length === 0) {
      return { Response: "No data returned" };
    }

    const payload = outputs.default[0];
    const responseData = payload?.data as Record<string, any> | undefined;

    const details: Record<string, string> = {};

    if (payload?.timestamp) {
      details["Deleted At"] = new Date(payload.timestamp).toLocaleString();
    }

    if (responseData?.id) {
      details["Check ID"] = String(responseData.id);
    }

    if (responseData?.deleted === true) {
      details["Status"] = "Deleted";
    }

    return details;
  },

  subtitle(context: SubtitleContext): string {
    if (!context.execution.createdAt) return "";
    return formatTimeAgo(new Date(context.execution.createdAt));
  },
};

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as DeleteHttpSyntheticCheckConfiguration;

  if (configuration?.checkId) {
    const idPreview =
      configuration.checkId.length > 24 ? configuration.checkId.substring(0, 24) + "…" : configuration.checkId;
    metadata.push({ icon: "fingerprint", label: idPreview });
  }

  if (configuration?.dataset) {
    metadata.push({ icon: "database", label: configuration.dataset });
  }

  return metadata;
}

function baseEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName!);
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });

  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventSubtitle: formatTimeAgo(new Date(execution.createdAt!)),
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent!.id!,
    },
  ];
}
