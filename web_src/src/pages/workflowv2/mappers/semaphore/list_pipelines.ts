import {
  ComponentBaseMapper,
  ComponentBaseContext,
  SubtitleContext,
  ExecutionDetailsContext,
  OutputPayload,
  NodeInfo,
  ExecutionInfo,
} from "../types";
import { ComponentBaseProps, DEFAULT_EVENT_STATE_MAP, EventSection } from "@/ui/componentBase";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { MetadataItem } from "@/ui/metadataList";
import { getTriggerRenderer } from "..";
import SemaphoreLogo from "@/assets/semaphore-logo-sign-black.svg";
import { formatTimeAgo } from "@/utils/date";

interface PipelineItem {
  ppl_id?: string;
  name?: string;
  wf_id?: string;
  state?: string;
  result?: string;
}

export const listPipelinesMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const { nodes, node, componentDefinition, lastExecutions } = context;
    return {
      title: node.name || componentDefinition.label || componentDefinition.name || "Unnamed component",
      iconSrc: SemaphoreLogo,
      iconSlug: componentDefinition.icon || "list",
      iconColor: getColorClass(componentDefinition?.color || "gray"),
      collapsed: node.isCollapsed,
      collapsedBackground: getBackgroundColorClass("white"),
      eventSections: listPipelinesEventSections(nodes, lastExecutions[0]),
      includeEmptyState: !lastExecutions[0],
      metadata: listPipelinesMetadataList(node),
      specs: [],
      eventStateMap: DEFAULT_EVENT_STATE_MAP,
    };
  },
  subtitle(context: SubtitleContext): string {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    if (outputs?.default?.[0]?.data) {
      const data = outputs.default[0].data;
      if (Array.isArray(data)) {
        const count = data.length;
        const timeStr = formatTimeAgo(new Date(context.execution.updatedAt || context.execution.createdAt || ""));
        return `${count} pipeline${count !== 1 ? "s" : ""} \u00b7 ${timeStr}`;
      }
    }
    const timestamp = context.execution.updatedAt || context.execution.createdAt;
    return timestamp ? formatTimeAgo(new Date(timestamp)) : "";
  },
  getExecutionDetails(context: ExecutionDetailsContext): Record<string, any> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const details: Record<string, any> = {};

    if (outputs?.default?.[0]?.data) {
      const data = outputs.default[0].data;
      if (Array.isArray(data)) {
        const pipelines = data as PipelineItem[];
        details["Total Pipelines"] = pipelines.length.toString();

        const passed = pipelines.filter((p) => p.result === "passed").length;
        const failed = pipelines.filter((p) => p.result === "failed").length;
        const running = pipelines.filter((p) => p.state === "running").length;

        if (passed > 0) details["Passed"] = passed.toString();
        if (failed > 0) details["Failed"] = failed.toString();
        if (running > 0) details["Running"] = running.toString();
      }
    }

    return details;
  },
};

function listPipelinesMetadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as any;

  if (configuration?.project) {
    metadata.push({ icon: "folder", label: configuration.project });
  }

  if (configuration?.branchName) {
    metadata.push({ icon: "git-branch", label: configuration.branchName });
  }

  return metadata;
}

function listPipelinesEventSections(
  nodes: NodeInfo[],
  execution: ExecutionInfo,
): EventSection[] | undefined {
  if (!execution) {
    return undefined;
  }

  const sections: EventSection[] = [];

  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName!);
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent! });

  let eventState: "neutral" | "success" | "error" | "cancelled" = "neutral";
  if (execution.resultReason === "RESULT_REASON_ERROR") {
    eventState = "error";
  } else if (execution.result === "RESULT_CANCELLED") {
    eventState = "cancelled";
  } else if (execution.state === "STATE_COMPLETED" && execution.result === "RESULT_PASSED") {
    eventState = "success";
  }

  const subtitleTimestamp = execution.updatedAt || execution.createdAt;
  const eventSubtitle = subtitleTimestamp ? formatTimeAgo(new Date(subtitleTimestamp)) : undefined;

  sections.push({
    receivedAt: new Date(execution.createdAt!),
    eventTitle: title,
    eventSubtitle,
    eventState,
    eventId: execution.rootEvent!.id!,
  });

  return sections;
}
