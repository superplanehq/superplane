import {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../../types";
import { ComponentBaseProps, ComponentBaseSpec, EventSection } from "@/ui/componentBase";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { getState, getStateMap, getTriggerRenderer } from "../..";
import { MetadataItem } from "@/ui/metadataList";
import { formatTimeAgo } from "@/utils/date";
import { stringOrDash } from "../../utils";
import awsCodePipelineIcon from "@/assets/icons/integrations/aws.codepipeline.svg";

interface RunPipelineConfiguration {
  region?: string;
  pipeline?: string;
}

interface RunPipelineMetadata {
  pipeline?: {
    name?: string;
    region?: string;
  };
}

interface RunPipelineOutput {
  pipeline?: {
    name?: string;
    executionId?: string;
    status?: string;
  };
}

export const runPipelineMapper: ComponentBaseMapper = {
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
      specs: getSpecs(context.node),
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const passedOutputs = context.execution.outputs as { passed?: OutputPayload[] } | undefined;
    const failedOutputs = context.execution.outputs as { failed?: OutputPayload[] } | undefined;
    const result =
      (passedOutputs?.passed?.[0]?.data as RunPipelineOutput | undefined) ||
      (failedOutputs?.failed?.[0]?.data as RunPipelineOutput | undefined);

    const details: Record<string, string> = {
      "Started At": context.execution.createdAt ? new Date(context.execution.createdAt).toLocaleString() : "-",
    };

    if (!result?.pipeline) {
      return details;
    }

    details["Pipeline"] = stringOrDash(result.pipeline.name);
    details["Execution ID"] = stringOrDash(result.pipeline.executionId);
    details["Status"] = stringOrDash(result.pipeline.status);

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
  const nodeMetadata = node.metadata as RunPipelineMetadata | undefined;
  const configuration = node.configuration as RunPipelineConfiguration | undefined;

  const pipelineName = nodeMetadata?.pipeline?.name || configuration?.pipeline;
  if (pipelineName) {
    metadata.push({ icon: "play", label: pipelineName });
  }

  const region = nodeMetadata?.pipeline?.region || configuration?.region;
  if (region) {
    metadata.push({ icon: "globe", label: region });
  }

  return metadata;
}

function getSpecs(node: NodeInfo): ComponentBaseSpec[] {
  const specs: ComponentBaseSpec[] = [];
  const configuration = node.configuration as RunPipelineConfiguration | undefined;

  if (configuration?.pipeline) {
    specs.push({
      title: "pipeline",
      tooltipTitle: "pipeline",
      iconSlug: "play",
      value: configuration.pipeline,
      contentType: "text",
    });
  }

  return specs;
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
