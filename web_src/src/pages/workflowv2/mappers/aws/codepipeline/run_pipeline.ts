import {
  ComponentBaseContext,
  ComponentBaseMapper,
  EventStateRegistry,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  StateFunction,
  SubtitleContext,
} from "../../types";
import {
  ComponentBaseProps,
  DEFAULT_EVENT_STATE_MAP,
  EventSection,
  EventState,
  EventStateMap,
} from "@/ui/componentBase";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { getTriggerRenderer } from "../..";
import { MetadataItem } from "@/ui/metadataList";
import { formatTimeAgo } from "@/utils/date";
import { stringOrDash } from "../../utils";
import { defaultStateFunction } from "../../stateRegistry";
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

export const RUN_PIPELINE_STATE_MAP: EventStateMap = {
  ...DEFAULT_EVENT_STATE_MAP,
  passed: DEFAULT_EVENT_STATE_MAP.success,
  failed: {
    icon: "circle-x",
    textColor: "text-gray-800",
    backgroundColor: "bg-red-100",
    badgeColor: "bg-red-400",
  },
};

export const runPipelineStateFunction: StateFunction = (execution: ExecutionInfo): EventState => {
  if (!execution) return "neutral";

  const outputs = execution.outputs as { failed?: OutputPayload[] } | undefined;
  if (outputs?.failed && outputs.failed.length > 0) {
    return "failed";
  }

  const state = defaultStateFunction(execution);
  return state === "success" ? "passed" : state;
};

export const RUN_PIPELINE_STATE_REGISTRY: EventStateRegistry = {
  stateMap: RUN_PIPELINE_STATE_MAP,
  getState: runPipelineStateFunction,
};

export const runPipelineMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;

    return {
      title: context.node.name || context.componentDefinition.label || "Unnamed component",
      iconSrc: awsCodePipelineIcon,
      iconColor: getColorClass(context.componentDefinition.color),
      collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
      collapsed: context.node.isCollapsed,
      eventSections: lastExecution ? getEventSections(context.nodes, lastExecution) : undefined,
      includeEmptyState: !lastExecution,
      metadata: getMetadataList(context.node),
      eventStateMap: RUN_PIPELINE_STATE_MAP,
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

function getEventSections(nodes: NodeInfo[], execution: ExecutionInfo): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName ?? "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });

  return [
    {
      receivedAt: new Date(execution.createdAt ?? 0),
      eventTitle: title,
      eventSubtitle: formatTimeAgo(new Date(execution.createdAt ?? 0)),
      eventState: runPipelineStateFunction(execution),
      eventId: execution.rootEvent?.id ?? "",
    },
  ];
}
