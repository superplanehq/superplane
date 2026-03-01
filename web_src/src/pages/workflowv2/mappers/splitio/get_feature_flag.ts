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
import splitioIcon from "@/assets/icons/integrations/splitio.svg";
import { buildSubtitle } from "../utils";
import { formatTimeAgo } from "@/utils/date";

interface GetFeatureFlagConfiguration {
  workspaceId?: string;
  environmentId?: string;
  flagName?: string;
}

interface FeatureFlagOutput {
  name?: string;
  killed?: boolean;
  defaultTreatment?: string;
  trafficAllocation?: number;
  creationTime?: number;
  lastUpdateTime?: number;
  workspaceId?: string;
  environmentId?: string;
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

function getFeatureFlagMetadata(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as GetFeatureFlagConfiguration | undefined;

  if (configuration?.flagName) {
    metadata.push({ icon: "flag", label: configuration.flagName });
  }

  return metadata;
}

export const getFeatureFlagMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const { node, componentDefinition, lastExecutions } = context;
    const lastExecution = lastExecutions.length > 0 ? lastExecutions[0] : null;
    const componentName = componentDefinition.name || node.componentName || "unknown";

    return {
      iconSrc: splitioIcon,
      iconColor: getColorClass(componentDefinition.color),
      collapsedBackground: getBackgroundColorClass(componentDefinition.color),
      collapsed: node.isCollapsed,
      title: node.name || componentDefinition.label || "Get Feature Flag",
      metadata: getFeatureFlagMetadata(node),
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

    const flag = outputs.default[0].data as FeatureFlagOutput;
    if (!flag) return details;

    if (flag.name) details["Name"] = flag.name;
    if (flag.defaultTreatment) details["Default Treatment"] = flag.defaultTreatment;
    if (flag.killed !== undefined) details["Killed"] = flag.killed ? "Yes" : "No";
    if (flag.trafficAllocation !== undefined) details["Traffic Allocation"] = `${flag.trafficAllocation}%`;
    if (flag.creationTime) details["Created At"] = new Date(flag.creationTime).toLocaleString();
    if (flag.lastUpdateTime) details["Last Updated"] = new Date(flag.lastUpdateTime).toLocaleString();

    return details;
  },
};
