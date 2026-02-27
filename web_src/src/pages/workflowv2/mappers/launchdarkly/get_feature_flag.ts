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

interface GetFeatureFlagConfiguration {
  projectKey?: string;
  flagKey?: string;
}

interface FeatureFlagOutput {
  projectKey?: string;
  key?: string;
  name?: string;
  description?: string;
  kind?: string;
  archived?: boolean;
  temporary?: boolean;
  creationDate?: number;
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

  if (configuration?.projectKey) {
    metadata.push({ icon: "folder", label: configuration.projectKey });
  }

  if (configuration?.flagKey) {
    metadata.push({ icon: "flag", label: configuration.flagKey });
  }

  return metadata;
}

export const getFeatureFlagMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const { node, componentDefinition, lastExecutions } = context;
    const lastExecution = lastExecutions.length > 0 ? lastExecutions[0] : null;
    const componentName = componentDefinition.name || node.componentName || "unknown";

    return {
      iconSrc: launchdarklyIcon,
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

    if (flag.projectKey) details["Project"] = flag.projectKey;
    if (flag.key) details["Key"] = flag.key;
    if (flag.name) details["Name"] = flag.name;
    if (flag.description) details["Description"] = flag.description;
    if (flag.kind) details["Kind"] = flag.kind;
    if (flag.archived !== undefined) details["Archived"] = flag.archived ? "Yes" : "No";
    if (flag.temporary !== undefined) details["Temporary"] = flag.temporary ? "Yes" : "No";
    if (flag.creationDate) details["Created At"] = new Date(flag.creationDate).toLocaleString();
    if (flag.projectKey && flag.key) {
      details["URL"] = `https://app.launchdarkly.com/projects/${flag.projectKey}/flags/${flag.key}`;
    }

    return details;
  },
};
