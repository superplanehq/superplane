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
    const componentName = context.componentDefinition.name || "semaphore.getPipeline";

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
    addDetail("Result Reason", payload.result_reason);
    addDetail("Branch", payload.branch_name);
    addDetail("Commit SHA", payload.commit_sha);
    addDetail("Commit Message", payload.commit_message);
    addDetail("YAML File", payload.yaml_file_name);
    addDetail("Working Directory", payload.working_directory);
    addDetail("Project ID", payload.project_id);
    addDetail("Created At", payload.created_at);
    addDetail("Done At", payload.done_at);
    addDetail("Running At", payload.running_at);
    addDetail("Error", payload.error_description);

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

function getPipelineEventSections(
  nodes: NodeInfo[],
  execution: ExecutionInfo,
  componentName: string,
): EventSection[] | undefined {
  // Return undefined if no root event
  if (!execution.rootEvent || !execution.rootEvent.id) {
    return undefined;
  }

  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName || "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({
    event: execution.rootEvent,
  });

  // Get state using the component-specific state function
  const executionState = getState(componentName)(execution);

  // Use updatedAt for subtitle when execution is complete, createdAt when running
  const subtitleTimestamp =
    executionState === "running" ? execution.createdAt : execution.updatedAt || execution.createdAt;

  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventSubtitle: subtitleTimestamp ? formatTimeAgo(new Date(subtitleTimestamp)) : "",
      eventState: executionState,
      eventId: execution.rootEvent.id,
    },
  ];
}
