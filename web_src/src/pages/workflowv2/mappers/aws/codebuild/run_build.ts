import {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../../types";
import { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { getState, getStateMap, getTriggerRenderer } from "../..";
import awsIcon from "@/assets/icons/integrations/aws.svg";
import { formatTimeAgo } from "@/utils/date";
import { formatTimestampInUserTimezone } from "@/utils/timezone";
import { MetadataItem } from "@/ui/metadataList";
import { CodeBuildBuildOutput, CodeBuildConfiguration, CodeBuildTriggerMetadata } from "./types";
import { buildProjectMetadataItems } from "./utils";
import { stringOrDash } from "../../utils";

interface RunBuildMetadata {
  projectName?: string;
}

export const runBuildMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name || "unknown";

    return {
      title: context.node.name || context.componentDefinition.label || "Unnamed component",
      iconSrc: awsIcon,
      iconColor: getColorClass(context.componentDefinition.color),
      collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
      collapsed: context.node.isCollapsed,
      eventSections: lastExecution ? getBuildEventSections(context.nodes, lastExecution, componentName) : undefined,
      includeEmptyState: !lastExecution,
      metadata: getBuildMetadataList(context.node),
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const build = outputs?.default?.[0]?.data as CodeBuildBuildOutput | undefined;
    if (!build) {
      return {};
    }

    return {
      Project: stringOrDash(build.projectName),
      "Build Status": stringOrDash(build.buildStatus),
      "Current Phase": stringOrDash(build.currentPhase),
      "Build Number": stringOrDash(build.buildNumber),
      "Build ID": stringOrDash(build.id),
      "Build ARN": stringOrDash(build.arn),
      "Source Version": stringOrDash(build.sourceVersion),
      Initiator: stringOrDash(build.initiator),
      "Started At": build.startTime ? formatTimestampInUserTimezone(build.startTime) : "-",
      "Finished At": build.endTime ? formatTimestampInUserTimezone(build.endTime) : "-",
      "Logs Link": stringOrDash(build.logs?.deepLink),
    };
  },

  subtitle(context: SubtitleContext): string {
    if (!context.execution.createdAt) {
      return "";
    }

    return formatTimeAgo(new Date(context.execution.createdAt));
  },
};

function getBuildMetadataList(node: NodeInfo): MetadataItem[] {
  const metadata = node.metadata as CodeBuildTriggerMetadata | RunBuildMetadata | undefined;
  const configuration = node.configuration as CodeBuildConfiguration | undefined;
  const metadataProjectName =
    metadata && typeof metadata === "object" && "projectName" in metadata ? metadata.projectName : undefined;

  return buildProjectMetadataItems(metadata as CodeBuildTriggerMetadata, configuration, metadataProjectName);
}

function getBuildEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
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
