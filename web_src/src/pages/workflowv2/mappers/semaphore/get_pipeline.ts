import {
  ComponentsComponent,
  ComponentsNode,
  CanvasesCanvasNodeExecution,
  CanvasesCanvasNodeQueueItem,
} from "@/api-client";
import { ComponentBaseMapper, OutputPayload } from "../types";
import { ComponentBaseProps, DEFAULT_EVENT_STATE_MAP, EventSection } from "@/ui/componentBase";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { MetadataItem } from "@/ui/metadataList";
import { getTriggerRenderer } from "..";
import SemaphoreLogo from "@/assets/semaphore-logo-sign-black.svg";
import { formatTimeAgo } from "@/utils/date";

interface PipelineOutput {
  name?: string;
  ppl_id?: string;
  wf_id?: string;
  state?: string;
  result?: string;
}

export const getPipelineMapper: ComponentBaseMapper = {
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
      iconSlug: componentDefinition.icon || "workflow",
      iconColor: getColorClass(componentDefinition?.color || "gray"),
      collapsed: node.isCollapsed,
      collapsedBackground: getBackgroundColorClass("white"),
      eventSections: getPipelineEventSections(nodes, lastExecutions[0]),
      includeEmptyState: !lastExecutions[0],
      metadata: getPipelineMetadataList(node),
      specs: [],
      eventStateMap: DEFAULT_EVENT_STATE_MAP,
    };
  },
  subtitle(_node: ComponentsNode, execution: CanvasesCanvasNodeExecution): string {
    const timestamp = execution.updatedAt || execution.createdAt;
    return timestamp ? formatTimeAgo(new Date(timestamp)) : "";
  },
  getExecutionDetails(execution: CanvasesCanvasNodeExecution, _node: ComponentsNode): Record<string, any> {
    const outputs = execution.outputs as { default?: OutputPayload[] } | undefined;
    const details: Record<string, any> = {};

    if (outputs?.default?.[0]?.data) {
      const pipeline = outputs.default[0].data as PipelineOutput;

      if (pipeline.ppl_id) {
        details["Pipeline ID"] = pipeline.ppl_id;
      }
      if (pipeline.name) {
        details["Name"] = pipeline.name;
      }
      if (pipeline.wf_id) {
        details["Workflow ID"] = pipeline.wf_id;
      }
      if (pipeline.state) {
        details["State"] = pipeline.state;
      }
      if (pipeline.result) {
        details["Result"] = pipeline.result;
      }
    }

    return details;
  },
};

function getPipelineMetadataList(node: ComponentsNode): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as any;

  if (configuration?.pipelineId) {
    const id = configuration.pipelineId as string;
    const display = id.length > 12 ? id.slice(0, 8) + "..." : id;
    metadata.push({ icon: "hash", label: `Pipeline: ${display}` });
  }

  return metadata;
}

function getPipelineEventSections(
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
