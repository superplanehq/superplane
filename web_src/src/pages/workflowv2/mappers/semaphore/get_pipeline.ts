import { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import { getBackgroundColorClass } from "@/utils/colors";
import { getState, getStateMap, getTriggerRenderer } from "..";
import {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import { MetadataItem } from "@/ui/metadataList";
import { stringOrDash } from "../utils";
import SemaphoreLogo from "@/assets/semaphore-logo-sign-black.svg";
import { formatTimeAgo } from "@/utils/date";

interface PipelineData {
  name?: string;
  ppl_id?: string;
  wf_id?: string;
  state?: string;
  result?: string;
  result_reason?: string;
  created_at?: { seconds?: number; nanos?: number } | string;
  done_at?: { seconds?: number; nanos?: number } | string;
  running_at?: { seconds?: number; nanos?: number } | string;
  pending_at?: { seconds?: number; nanos?: number } | string;
  queuing_at?: { seconds?: number; nanos?: number } | string;
  stopping_at?: { seconds?: number; nanos?: number } | string;
  yaml_file_name?: string;
  working_directory?: string;
  error_description?: string;
}

function formatTimestamp(value?: { seconds?: number; nanos?: number } | string): string | undefined {
  if (!value) return undefined;

  if (typeof value === "string") {
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) return undefined;
    return date.toLocaleString();
  }

  if (typeof value === "object" && value.seconds) {
    const date = new Date(value.seconds * 1000);
    if (Number.isNaN(date.getTime())) return undefined;
    return date.toLocaleString();
  }

  return undefined;
}

export const getPipelineMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name || "unknown";

    return {
      iconSrc: SemaphoreLogo,
      collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
      collapsed: context.node.isCollapsed,
      title:
        context.node.name ||
        context.componentDefinition.label ||
        context.componentDefinition.name ||
        "Unnamed component",
      eventSections: lastExecution ? getPipelineEventSections(context.nodes, lastExecution, componentName) : undefined,
      metadata: getPipelineMetadata(context.node),
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const details: Record<string, string> = {};
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;

    if (!outputs?.default || outputs.default.length === 0) {
      return details;
    }

    const pipeline = outputs.default[0].data as PipelineData;
    if (!pipeline || typeof pipeline !== "object") {
      return details;
    }

    details["Pipeline Name"] = stringOrDash(pipeline.name);
    details["Pipeline ID"] = stringOrDash(pipeline.ppl_id);
    details["Workflow ID"] = stringOrDash(pipeline.wf_id);
    details["State"] = stringOrDash(pipeline.state);
    details["Result"] = stringOrDash(pipeline.result);

    const doneAt = formatTimestamp(pipeline.done_at);
    if (doneAt) {
      details["Done At"] = doneAt;
    }

    const createdAt = formatTimestamp(pipeline.created_at);
    if (createdAt) {
      details["Created At"] = createdAt;
    }

    if (pipeline.yaml_file_name) {
      const pipelineFile = pipeline.working_directory
        ? `${pipeline.working_directory}/${pipeline.yaml_file_name}`.replace("//", "/")
        : pipeline.yaml_file_name;
      details["Pipeline File"] = pipelineFile;
    }

    if (pipeline.error_description) {
      details["Error"] = pipeline.error_description;
    }

    return details;
  },

  subtitle(context: SubtitleContext): string {
    if (!context.execution.createdAt) return "";
    return formatTimeAgo(new Date(context.execution.createdAt));
  },
};

function getPipelineMetadata(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as { pipelineId?: string };

  if (configuration?.pipelineId) {
    metadata.push({ icon: "git-branch", label: "Pipeline: " + configuration.pipelineId });
  }

  return metadata;
}

function getPipelineEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
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
