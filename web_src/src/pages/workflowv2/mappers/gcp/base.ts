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
import gcpIcon from "@/assets/icons/integrations/gcp.svg";
import cloudBuildIcon from "@/assets/icons/integrations/cloud_build.svg";
import { renderTimeAgo } from "@/components/TimeAgo";
import { buildCloudBuildSummaryDetails, getCloudBuildData, getCloudBuildOutputPayload } from "./cloudbuild";

export const baseMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name ?? "gcp";

    return {
      iconSrc: gcpIcon,
      iconSlug: context.componentDefinition?.icon ?? "cloud",
      collapsedBackground: "bg-white",
      collapsed: context.node.isCollapsed,
      title: context.node.name || context.componentDefinition?.label || context.componentDefinition?.name || "GCP",
      eventSections: lastExecution ? baseEventSections(context.nodes, lastExecution, componentName) : undefined,
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const details: Record<string, string> = {};
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const payload = outputs?.default?.[0];

    if (payload?.type) {
      details["Event Type"] = payload.type;
    }

    if (payload?.timestamp) {
      details["Emitted At"] = new Date(payload.timestamp).toLocaleString();
    }

    return details;
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    const timestamp = context.execution.updatedAt || context.execution.createdAt;
    return timestamp ? renderTimeAgo(new Date(timestamp)) : "";
  },
};

export const cloudBuildBaseMapper: ComponentBaseMapper = {
  ...baseMapper,
  props(context: ComponentBaseContext): ComponentBaseProps {
    const baseProps = baseMapper.props(context);
    return {
      ...baseProps,
      iconSrc: cloudBuildIcon,
      eventSections: baseProps.eventSections?.map((section) => ({
        ...section,
        showAutomaticTime: true,
      })),
    };
  },
  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const payload = getCloudBuildOutputPayload(context.execution);
    const build = getCloudBuildData(context.execution);
    const timestamp = payload?.timestamp ?? context.execution.updatedAt ?? context.execution.createdAt;
    return buildCloudBuildSummaryDetails({ build, timestamp });
  },
};

function baseEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName!);
  const { title, subtitle } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });
  const subtitleTimestamp = execution.updatedAt || execution.createdAt;
  const fallbackSubtitle = subtitleTimestamp ? renderTimeAgo(new Date(subtitleTimestamp)) : "";
  const eventSubtitle = subtitle || fallbackSubtitle;

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
