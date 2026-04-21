import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../../types";
import type { ComponentBaseProps, EventSection } from "@/pages/workflowv2/mappers/rendererTypes";
import type React from "react";
import { getBackgroundColorClass, getColorClass } from "@/lib/colors";
import { getState, getStateMap } from "../..";
import type { MetadataItem } from "@/ui/metadataList";
import { renderTimeAgo } from "@/components/TimeAgo";
import { stringOrDash } from "../../utils";
import awsCodePipelineIcon from "@/assets/icons/integrations/aws.codepipeline.svg";

interface RetryStageExecutionConfiguration {
  region?: string;
  pipeline?: string;
  stage?: string;
  pipelineExecution?: string;
  retryMode?: string;
}

interface RetryStageExecutionOutput {
  pipeline?: {
    name?: string;
    stage?: string;
    retryMode?: string;
    sourceExecutionId?: string;
    newExecutionId?: string;
  };
}

export const retryStageExecutionMapper: ComponentBaseMapper = {
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
    const result = outputs?.default?.[0]?.data as RetryStageExecutionOutput | undefined;

    const timestamp = context.execution.updatedAt
      ? new Date(context.execution.updatedAt).toLocaleString()
      : context.execution.createdAt
        ? new Date(context.execution.createdAt).toLocaleString()
        : "-";

    const details: Record<string, string> = {
      Timestamp: timestamp,
    };

    if (!result?.pipeline) {
      return details;
    }

    details["Pipeline"] = stringOrDash(result.pipeline.name);
    details["Stage"] = stringOrDash(result.pipeline.stage);
    details["Retry Mode"] = stringOrDash(result.pipeline.retryMode);
    details["Source Execution ID"] = stringOrDash(result.pipeline.sourceExecutionId);
    details["New Execution ID"] = stringOrDash(result.pipeline.newExecutionId);

    return details;
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    if (!context.execution.createdAt) {
      return "";
    }
    return renderTimeAgo(new Date(context.execution.createdAt));
  },
};

function getMetadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as RetryStageExecutionConfiguration | undefined;

  if (configuration?.pipeline) {
    metadata.push({ icon: "file-text", label: configuration.pipeline });
  }

  if (configuration?.stage) {
    metadata.push({ icon: "layers", label: configuration.stage });
  }

  if (configuration?.region) {
    metadata.push({ icon: "globe", label: configuration.region });
  }

  return metadata;
}

function getEventSections(_nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  return [
    {
      receivedAt: new Date(execution.createdAt ?? 0),
      eventSubtitle: renderTimeAgo(new Date(execution.createdAt ?? 0)),
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent?.id ?? "",
    },
  ];
}
