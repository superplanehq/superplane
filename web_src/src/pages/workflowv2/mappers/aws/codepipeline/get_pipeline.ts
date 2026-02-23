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
import { MetadataItem } from "@/ui/metadataList";
import { formatTimeAgo } from "@/utils/date";
import { stringOrDash } from "../../utils";
import awsCodePipelineIcon from "@/assets/icons/integrations/aws.codepipeline.svg";

interface GetPipelineConfiguration {
  region?: string;
  pipeline?: string;
}

interface GetPipelineOutput {
  pipeline?: {
    name?: string;
    roleArn?: string;
    version?: number;
    stages?: unknown[];
  };
  metadata?: {
    pipelineArn?: string;
    created?: string;
    updated?: string;
  };
}

export const getPipelineMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name || "unknown";

    return {
      title: context.node.name || context.componentDefinition.label || "Unnamed component",
      iconSrc: awsCodePipelineIcon,
      iconColor: getColorClass(context.componentDefinition.color),
      collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
      collapsed: context.node.isCollapsed,
      eventSections: lastExecution ? getEventSections(context.nodes, lastExecution, componentName) : undefined,
      includeEmptyState: !lastExecution,
      metadata: getMetadataList(context.node),
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const result = outputs?.default?.[0]?.data as GetPipelineOutput | undefined;

    const timestamp = context.execution.updatedAt
      ? new Date(context.execution.updatedAt).toLocaleString()
      : context.execution.createdAt
        ? new Date(context.execution.createdAt).toLocaleString()
        : "-";

    const details: Record<string, string> = {
      "Retrieved At": timestamp,
    };

    if (result?.pipeline) {
      details["Pipeline"] = stringOrDash(result.pipeline.name);
      details["Version"] = result.pipeline.version ? String(result.pipeline.version) : "-";
      details["Stages"] = result.pipeline.stages ? String(result.pipeline.stages.length) : "-";
    }

    if (result?.metadata?.pipelineArn) {
      details["Pipeline ARN"] = result.metadata.pipelineArn;
    }

    return details;
  },

  subtitle(context: SubtitleContext): string {
    if (!context.execution.createdAt) {
      return "";
    }
    return formatTimeAgo(new Date(context.execution.createdAt));
  },
};

function getMetadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as GetPipelineConfiguration | undefined;

  if (configuration?.pipeline) {
    metadata.push({ icon: "file-text", label: configuration.pipeline });
  }

  if (configuration?.region) {
    metadata.push({ icon: "globe", label: configuration.region });
  }

  return metadata;
}

function getEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName ?? "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });

  return [
    {
      receivedAt: new Date(execution.createdAt ?? 0),
      eventTitle: title,
      eventSubtitle: formatTimeAgo(new Date(execution.createdAt ?? 0)),
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent?.id ?? "",
    },
  ];
}
