import {
  ComponentsComponent,
  ComponentsNode,
  CanvasesCanvasNodeExecution,
  CanvasesCanvasNodeQueueItem,
} from "@/api-client";
import { ComponentBaseMapper, OutputPayload } from "../types";
import {
  ComponentBaseProps,
  DEFAULT_EVENT_STATE_MAP,
  EventSection,
} from "@/ui/componentBase";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { MetadataItem } from "@/ui/metadataList";
import { getTriggerRenderer } from "..";
import SemaphoreLogo from "@/assets/semaphore-logo-sign-black.svg";
import { formatTimeAgo } from "@/utils/date";

interface JobLogsOutput {
  events?: Array<{
    event?: string;
    timestamp?: number;
    output?: string;
    directive?: string;
    exit_code?: number;
    started_at?: number;
    finished_at?: number;
    result?: string;
  }>;
  output?: string;
  lineCount?: number;
  result?: string;
}

export const getJobLogsMapper: ComponentBaseMapper = {
  props(
    nodes: ComponentsNode[],
    node: ComponentsNode,
    componentDefinition: ComponentsComponent,
    lastExecutions: CanvasesCanvasNodeExecution[],
    _nodeQueueItems?: CanvasesCanvasNodeQueueItem[],
  ): ComponentBaseProps {
    return {
      title: node.name || componentDefinition.label || componentDefinition.name || "Unnamed component",
      iconSrc: SemaphoreLogo,
      iconSlug: componentDefinition.icon || "file-text",
      iconColor: getColorClass(componentDefinition?.color || "gray"),
      collapsed: node.isCollapsed,
      collapsedBackground: getBackgroundColorClass("white"),
      eventSections: getJobLogsEventSections(nodes, lastExecutions[0]),
      includeEmptyState: !lastExecutions[0],
      metadata: getJobLogsMetadataList(node),
      specs: [],
      eventStateMap: DEFAULT_EVENT_STATE_MAP,
    };
  },
  subtitle(_node: ComponentsNode, execution: CanvasesCanvasNodeExecution): string {
    const timestamp = execution.updatedAt || execution.createdAt;
    return timestamp ? formatTimeAgo(new Date(timestamp)) : "";
  },
  getExecutionDetails(execution: CanvasesCanvasNodeExecution, _node: ComponentsNode): Record<string, any> {
    const details: Record<string, any> = {};
    const outputs = execution.outputs as { default?: OutputPayload[] } | undefined;

    if (outputs?.default?.[0]?.data) {
      const logsOutput = outputs.default[0].data as JobLogsOutput;

      if (logsOutput.result) {
        details["Job Result"] = logsOutput.result;
      }

      if (logsOutput.lineCount !== undefined) {
        details["Line Count"] = logsOutput.lineCount.toString();
      }

      if (logsOutput.output) {
        details["Output"] = {
          __type: "codeBlock",
          content: logsOutput.output,
          language: "text",
        };
      }
    }

    return details;
  },
};

function getJobLogsMetadataList(node: ComponentsNode): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as any;

  if (configuration?.jobId) {
    metadata.push({ icon: "hash", label: `Job: ${configuration.jobId}` });
  }

  if (configuration?.limit) {
    metadata.push({ icon: "list", label: `Limit: ${configuration.limit}` });
  }

  return metadata;
}

function getJobLogsEventSections(
  nodes: ComponentsNode[],
  execution: CanvasesCanvasNodeExecution,
): EventSection[] | undefined {
  if (!execution) {
    return undefined;
  }

  const sections: EventSection[] = [];

  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.trigger?.name || "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle(execution.rootEvent!);

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
