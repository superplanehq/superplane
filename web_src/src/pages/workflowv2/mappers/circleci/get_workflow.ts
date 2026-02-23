import {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { getState, getStateMap, getTriggerRenderer } from "..";
import { MetadataItem } from "@/ui/metadataList";
import { formatTimeAgo } from "@/utils/date";
import CircleCILogo from "@/assets/icons/integrations/circleci.svg";

interface GetWorkflowOutput {
  workflow?: {
    id?: string;
    name?: string;
    status?: string;
    created_at?: string;
    stopped_at?: string;
  };
  jobs?: Array<{
    id?: string;
    name?: string;
    status?: string;
    job_number?: number;
  }>;
}

export const getWorkflowMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name || "unknown";

    return {
      title: context.node.name || context.componentDefinition.label || "Unnamed component",
      iconSrc: CircleCILogo,
      iconColor: getColorClass(context.componentDefinition.color),
      collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
      collapsed: context.node.isCollapsed,
      eventSections: lastExecution ? getEventSections(context.nodes, lastExecution, componentName) : undefined,
      includeEmptyState: !lastExecution,
      metadata: getMetadataList(context.node),
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const result = outputs?.default?.[0]?.data as GetWorkflowOutput | undefined;

    const details: Record<string, string> = {};

    if (result?.workflow) {
      if (result.workflow.name) details["Workflow"] = result.workflow.name;
      if (result.workflow.status) details["Status"] = result.workflow.status;
      if (result.workflow.id) details["Workflow ID"] = result.workflow.id;
    }

    if (result?.jobs) {
      details["Jobs"] = String(result.jobs.length);
    }

    const timestamp = context.execution.updatedAt || context.execution.createdAt;
    if (timestamp) {
      details["Retrieved At"] = new Date(timestamp).toLocaleString();
    }

    return details;
  },

  subtitle(context: SubtitleContext): string {
    if (!context.execution.createdAt) return "";
    return formatTimeAgo(new Date(context.execution.createdAt));
  },
};

function getMetadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as { workflowId?: string } | undefined;

  if (configuration?.workflowId) {
    metadata.push({ icon: "workflow", label: configuration.workflowId });
  }

  return metadata;
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
