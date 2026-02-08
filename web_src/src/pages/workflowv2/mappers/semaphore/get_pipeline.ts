import {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  SubtitleContext,
} from "../types";
import { ComponentBaseProps, ComponentBaseSpec, EventSection } from "@/ui/componentBase";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { getState, getStateMap, getTriggerRenderer } from "..";
import { MetadataItem } from "@/ui/metadataList";
import SemaphoreLogo from "@/assets/semaphore-logo-sign-black.svg";
import { formatTimeAgo } from "@/utils/date";

interface GetPipelineConfiguration {
  pipelineId?: string;
}

export const getPipelineMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name || "unknown";

    return {
      title:
        context.node.name ||
        context.componentDefinition.label ||
        context.componentDefinition.name ||
        "Unnamed component",
      iconSrc: SemaphoreLogo,
      iconSlug: context.componentDefinition.icon || "workflow",
      iconColor: getColorClass(context.componentDefinition?.color || "gray"),
      collapsed: context.node.isCollapsed,
      collapsedBackground: getBackgroundColorClass("white"),
      includeEmptyState: !lastExecution,
      metadata: getPipelineMetadataList(context.node),
      specs: getPipelineSpecs(context.node),
      eventSections: lastExecution ? getPipelineEventSections(context.nodes, lastExecution, componentName) : undefined,
      eventStateMap: getStateMap(componentName),
    };
  },
  subtitle(context: SubtitleContext): string {
    const timestamp = context.execution.updatedAt || context.execution.createdAt;
    return timestamp ? formatTimeAgo(new Date(timestamp)) : "";
  },
  getExecutionDetails(context: ExecutionDetailsContext): Record<string, any> {
    const details: Record<string, any> = {};
    const outputs = context.execution.outputs as { default?: { data?: any }[] } | undefined;
    const payload = outputs?.default?.[0]?.data as Record<string, any> | undefined;

    if (!payload || typeof payload !== "object") {
      return details;
    }

    const addDetail = (key: string, value?: string) => {
      if (value) {
        details[key] = value;
      }
    };

    addDetail("Pipeline ID", payload.ppl_id);
    addDetail("Pipeline Name", payload.name);
    addDetail("Workflow ID", payload.wf_id);
    addDetail("State", payload.state);
    addDetail("Result", payload.result);

    return details;
  },
};

function getPipelineMetadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as GetPipelineConfiguration | undefined;

  if (configuration?.pipelineId) {
    metadata.push({ icon: "hash", label: configuration.pipelineId });
  }

  return metadata;
}

function getPipelineSpecs(_node: NodeInfo): ComponentBaseSpec[] {
  return [];
}

function getPipelineEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName!);
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({
    event: execution.rootEvent,
  });

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
