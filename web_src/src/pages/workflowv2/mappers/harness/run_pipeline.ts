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
  ComponentBaseSpec,
  DEFAULT_EVENT_STATE_MAP,
  EventSection,
  EventState,
  EventStateMap,
} from "@/ui/componentBase";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { MetadataItem } from "@/ui/metadataList";
import HarnessIcon from "@/assets/icons/integrations/harness.svg";
import { formatTimeAgo } from "@/utils/date";
import { getTriggerRenderer } from "..";
import { CanvasesCanvasNodeExecution } from "@/api-client";

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
      iconSrc: HarnessIcon,
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

    const addDetail = (key: string, value?: string) => {
      if (value) details[key] = value;
    };

    addDetail("Execution ID", sourceData.executionId as string | undefined);
    addDetail("Pipeline", sourceData.pipelineIdentifier as string | undefined);
    addDetail("Status", sourceData.status as string | undefined);
    addDetail("Plan URL", sourceData.planExecutionUrl as string | undefined);
    addDetail("Started At", sourceData.startedAt as string | undefined);
    addDetail("Ended At", sourceData.endedAt as string | undefined);

    return details;
  },
};

function runPipelineMetadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as { pipelineIdentifier?: string; ref?: string };

  if (configuration?.pipelineIdentifier) {
    metadata.push({ icon: "workflow", label: configuration.pipelineIdentifier });
  }

  if (configuration?.ref) {
    metadata.push({ icon: "git-branch", label: configuration.ref });
  }

  return metadata;
}

function runPipelineSpecs(node: NodeInfo): ComponentBaseSpec[] {
  const specs: ComponentBaseSpec[] = [];
  const configuration = node.configuration as { inputSetReferences?: string[] };

  const inputSetReferences = configuration?.inputSetReferences;
  if (inputSetReferences && inputSetReferences.length > 0) {
    specs.push({
      title: "input set",
      tooltipTitle: "input set references",
      iconSlug: "list",
      values: inputSetReferences.map((inputSetReference) => ({
        badges: [
          {
            label: inputSetReference,
            bgColor: "bg-orange-100",
            textColor: "text-orange-900",
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

  const rootTriggerNode = nodes.find((node) => node.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName || "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });

  const subtitleTimestamp = execution.updatedAt || execution.createdAt;
  const eventSubtitle = subtitleTimestamp ? formatTimeAgo(new Date(subtitleTimestamp)) : "";
  const eventID = execution.rootEvent?.id || execution.id || "execution";

  return [
    {
      receivedAt: execution.createdAt ? new Date(execution.createdAt) : new Date(),
      eventTitle: title,
      eventSubtitle,
      eventState: runPipelineStateFunction(execution),
      eventId: eventID,
    },
  ];
}
