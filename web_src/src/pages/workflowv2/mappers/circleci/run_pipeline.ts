import { CanvasesCanvasNodeExecution } from "@/api-client";
import CircleCILogo from "@/assets/icons/integrations/circleci.svg";
import {
  ComponentBaseProps,
  ComponentBaseSpec,
  DEFAULT_EVENT_STATE_MAP,
  EventSection,
  EventState,
  EventStateMap,
} from "@/ui/componentBase";
import { MetadataItem } from "@/ui/metadataList";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { formatTimeAgo } from "@/utils/date";
import { getTriggerRenderer } from "..";
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
} from "../types";

export const RUN_PIPELINE_STATE_MAP: EventStateMap = {
  ...DEFAULT_EVENT_STATE_MAP,
  running: {
    icon: "loader-circle",
    textColor: "text-gray-800",
    backgroundColor: "bg-blue-100",
    badgeColor: "bg-blue-500",
  },
  success: {
    icon: "circle-check",
    textColor: "text-gray-800",
    backgroundColor: "bg-green-100",
    badgeColor: "bg-emerald-500",
  },
  failed: {
    icon: "circle-x",
    textColor: "text-gray-800",
    backgroundColor: "bg-red-100",
    badgeColor: "bg-red-400",
  },
};

export const runPipelineStateFunction: StateFunction = (execution: CanvasesCanvasNodeExecution): EventState => {
  if (!execution) return "neutral";

  if (
    execution.resultMessage &&
    (execution.resultReason === "RESULT_REASON_ERROR" ||
      (execution.result === "RESULT_FAILED" && execution.resultReason !== "RESULT_REASON_ERROR_RESOLVED"))
  ) {
    return "error";
  }

  if (execution.result === "RESULT_CANCELLED") {
    return "cancelled";
  }

  if (execution.state === "STATE_PENDING" || execution.state === "STATE_STARTED") {
    return "running";
  }

  const outputs = execution.outputs as { success?: OutputPayload[]; failed?: OutputPayload[] } | undefined;

  if (outputs?.failed && outputs.failed.length > 0) {
    return "failed";
  }

  if (outputs?.success && outputs.success.length > 0) {
    return "success";
  }

  return "neutral";
};

export const RUN_PIPELINE_STATE_REGISTRY: EventStateRegistry = {
  stateMap: RUN_PIPELINE_STATE_MAP,
  getState: runPipelineStateFunction,
};

export const runPipelineMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;

    return {
      title:
        context.node.name ||
        context.componentDefinition.label ||
        context.componentDefinition.name ||
        "Unnamed component",
      iconSrc: CircleCILogo,
      iconSlug: context.componentDefinition.icon || "workflow",
      iconColor: getColorClass(context.componentDefinition?.color || "gray"),
      collapsed: context.node.isCollapsed,
      collapsedBackground: getBackgroundColorClass("white"),
      eventSections: lastExecution ? runPipelineEventSections(context.nodes, lastExecution) : undefined,
      includeEmptyState: !lastExecution,
      metadata: runPipelineMetadataList(context.node),
      specs: runPipelineSpecs(context.node),
      eventStateMap: RUN_PIPELINE_STATE_MAP,
    };
  },
  subtitle(context: SubtitleContext): string {
    const timestamp = context.execution.updatedAt || context.execution.createdAt;
    return timestamp ? formatTimeAgo(new Date(timestamp)) : "";
  },
  getExecutionDetails(context: ExecutionDetailsContext): Record<string, any> {
    const details: Record<string, any> = {};
    const outputs = context.execution.outputs as
      | { success?: OutputPayload[]; failed?: OutputPayload[]; default?: OutputPayload[] }
      | undefined;
    const payload =
      (outputs?.success?.[0]?.data as Record<string, any> | undefined) ||
      (outputs?.failed?.[0]?.data as Record<string, any> | undefined) ||
      (outputs?.default?.[0]?.data as Record<string, any> | undefined);

    const payloadData =
      payload && typeof payload === "object" && payload.data && typeof payload.data === "object"
        ? payload.data
        : payload;

    const metadataFallback =
      (!payloadData || typeof payloadData !== "object") && context.execution.metadata
        ? (context.execution.metadata as Record<string, any>)
        : undefined;

    const sourceData =
      payloadData && typeof payloadData === "object"
        ? payloadData
        : metadataFallback && typeof metadataFallback === "object"
          ? metadataFallback
          : undefined;

    if (!sourceData || typeof sourceData !== "object") {
      return details;
    }

    const pipeline = sourceData.pipeline as Record<string, any> | undefined;

    const addDetail = (key: string, value?: string) => {
      if (value) {
        details[key] = value;
      }
    };

    addDetail("Started At", formatDate(context.execution.createdAt));
    addDetail("Finished At", formatDate(context.execution.updatedAt));
    addDetail("Pipeline Number", pipeline?.number?.toString());
    addDetail("Pipeline ID", pipeline?.id);
    addDetail("Pipeline URL", pipeline?.pipeline_url);

    return details;
  },
};

function runPipelineMetadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as any;
  const nodeMetadata = node.metadata as
    | {
        projectName?: string;
        pipelineDefinitionName?: string;
      }
    | undefined;

  const projectLabel = nodeMetadata?.projectName || configuration?.projectSlug;
  if (projectLabel) {
    metadata.push({ icon: "folder", label: projectLabel });
  }

  const pipelineName = nodeMetadata?.pipelineDefinitionName;
  if (pipelineName) {
    metadata.push({ icon: "workflow", label: pipelineName });
  }

  if (configuration?.branch) {
    metadata.push({ icon: "git-branch", label: configuration.branch });
  } else if (configuration?.tag) {
    metadata.push({ icon: "tag", label: configuration.tag });
  }

  return metadata;
}

function runPipelineSpecs(node: NodeInfo): ComponentBaseSpec[] {
  const specs: ComponentBaseSpec[] = [];
  const configuration = node.configuration as any;

  const parameters = configuration?.parameters as Array<{ name: string; value: string }> | undefined;
  if (parameters && parameters.length > 0) {
    specs.push({
      title: "parameter",
      tooltipTitle: "pipeline parameters",
      iconSlug: "settings",
      values: parameters.map((param) => ({
        badges: [
          {
            label: param.name,
            bgColor: "bg-purple-100",
            textColor: "text-purple-800",
          },
          {
            label: param.value,
            bgColor: "bg-gray-100",
            textColor: "text-gray-800",
          },
        ],
      })),
    });
  }

  return specs;
}

function runPipelineEventSections(nodes: NodeInfo[], execution: ExecutionInfo): EventSection[] | undefined {
  if (!execution) {
    return undefined;
  }

  const sections: EventSection[] = [];

  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName!);
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });
  const executionState = runPipelineStateFunction(execution);
  const subtitleTimestamp =
    executionState === "running" ? execution.createdAt : execution.updatedAt || execution.createdAt;
  const eventSubtitle = subtitleTimestamp ? formatTimeAgo(new Date(subtitleTimestamp)) : undefined;

  sections.push({
    receivedAt: new Date(execution.createdAt!),
    eventTitle: title,
    eventSubtitle,
    eventState: executionState,
    eventId: execution.rootEvent!.id!,
  });

  return sections;
}

function formatDate(value?: string): string | undefined {
  if (!value) return undefined;
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return undefined;
  return date.toLocaleString();
}
