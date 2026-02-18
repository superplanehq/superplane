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
import {
  ComponentBaseProps,
  DEFAULT_EVENT_STATE_MAP,
  EventSection,
  EventState,
  EventStateMap,
} from "@/ui/componentBase";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { MetadataItem } from "@/ui/metadataList";
import { getTriggerRenderer } from "..";
import { formatTimeAgo } from "@/utils/date";
import { CanvasesCanvasNodeExecution } from "@/api-client";

interface ExecutionMetadata {
  pipeline?: {
    planExecutionId: string;
    status: string;
    name: string;
    executionUrl: string;
  };
}

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

  const metadata = execution.metadata as ExecutionMetadata;
  const pipelineStatus = metadata?.pipeline?.status;
  if (pipelineStatus === "Success" || pipelineStatus === "IgnoreFailed") {
    return "success";
  }
  if (pipelineStatus === "Failed" || pipelineStatus === "Errored" || pipelineStatus === "Aborted") {
    return "failed";
  }

  return "success";
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
      iconSlug: context.componentDefinition.icon || "workflow",
      iconColor: getColorClass(context.componentDefinition?.color || "blue"),
      collapsed: context.node.isCollapsed,
      collapsedBackground: getBackgroundColorClass("white"),
      eventSections: lastExecution ? runPipelineEventSections(context.nodes, lastExecution) : undefined,
      includeEmptyState: !lastExecution,
      metadata: runPipelineMetadataList(context.node),
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

    const addDetail = (key: string, value?: string) => {
      if (value) {
        details[key] = value;
      }
    };

    addDetail("Pipeline", sourceData.name as string | undefined);
    addDetail("Status", sourceData.status as string | undefined);
    addDetail("Execution URL", sourceData.executionUrl as string | undefined);

    const startTs = sourceData.startTs as number | undefined;
    const endTs = sourceData.endTs as number | undefined;
    if (startTs) {
      addDetail("Started", new Date(startTs).toLocaleString());
    }
    if (endTs) {
      addDetail("Finished", new Date(endTs).toLocaleString());
    }

    return details;
  },
};

function runPipelineMetadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as any;

  if (configuration?.orgIdentifier) {
    metadata.push({ icon: "building", label: configuration.orgIdentifier });
  }

  if (configuration?.projectIdentifier) {
    metadata.push({ icon: "folder", label: configuration.projectIdentifier });
  }

  if (configuration?.pipelineIdentifier) {
    metadata.push({ icon: "git-branch", label: configuration.pipelineIdentifier });
  }

  if (configuration?.module) {
    metadata.push({ icon: "box", label: configuration.module });
  }

  return metadata;
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
