import { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import {
  ComponentBaseMapper,
  ComponentBaseContext,
  SubtitleContext,
  ExecutionDetailsContext,
  ExecutionInfo,
  OutputPayload,
  NodeInfo,
} from "../types";
import { MetadataItem } from "@/ui/metadataList";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { getState, getStateMap, getTriggerRenderer } from "..";
import launchdarklyIcon from "@/assets/icons/integrations/launchdarkly.svg";
import { buildSubtitle } from "../utils";
import { formatTimeAgo } from "@/utils/date";

interface DeleteFeatureFlagConfiguration {
  projectKey?: string;
  flagKey?: string;
}

interface DeleteFeatureFlagOutput {
  projectKey?: string;
  flagKey?: string;
  deleted?: boolean;
}

function getEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName!);
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });
  const subtitleTimestamp = execution.updatedAt || execution.createdAt;
  const eventSubtitle = subtitleTimestamp ? formatTimeAgo(new Date(subtitleTimestamp)) : "";
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

function deleteFeatureFlagMetadata(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as DeleteFeatureFlagConfiguration | undefined;

  if (configuration?.projectKey) {
    metadata.push({ icon: "folder", label: configuration.projectKey });
  }

  if (configuration?.flagKey) {
    metadata.push({ icon: "flag", label: configuration.flagKey });
  }

  return metadata;
}

export const deleteFeatureFlagMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const { node, componentDefinition, lastExecutions } = context;
    const lastExecution = lastExecutions.length > 0 ? lastExecutions[0] : null;
    const componentName = componentDefinition.name || node.componentName || "unknown";

    return {
      iconSrc: launchdarklyIcon,
      iconColor: getColorClass(componentDefinition.color),
      collapsedBackground: getBackgroundColorClass(componentDefinition.color),
      collapsed: node.isCollapsed,
      title: node.name || componentDefinition.label || "Delete Feature Flag",
      metadata: deleteFeatureFlagMetadata(node),
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
      eventSections: lastExecution ? getEventSections(context.nodes, lastExecution, componentName) : undefined,
    };
  },

  subtitle(context: SubtitleContext): string {
    return buildSubtitle("", context.execution.updatedAt || context.execution.createdAt);
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const details: Record<string, string> = {};

    if (!outputs?.default?.length) {
      return details;
    }

    const result = outputs.default[0].data as DeleteFeatureFlagOutput;
    if (!result) return details;

    if (result.projectKey) details["Project"] = result.projectKey;
    if (result.flagKey) details["Flag"] = result.flagKey;
    if (result.deleted !== undefined) details["Deleted"] = result.deleted ? "Yes" : "No";

    return details;
  },
};
