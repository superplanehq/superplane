import { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import { getColorClass, getBackgroundColorClass } from "@/utils/colors";
import { getState, getStateMap, getTriggerRenderer } from "..";
import terraformIcon from "@/assets/icons/integrations/terraform.svg";
import { MetadataItem } from "@/ui/metadataList";
import {
  OutputPayload,
  ComponentBaseMapper,
  ComponentBaseContext,
  SubtitleContext,
  ExecutionDetailsContext,
  NodeInfo,
  ExecutionInfo,
} from "../types";

export const terraformComponentMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const { nodes, node, componentDefinition, lastExecutions } = context;
    const lastExecution = lastExecutions.length > 0 ? lastExecutions[0] : null;
    const componentName = componentDefinition.name || node.componentName || "unknown";

    const metadata: MetadataItem[] = [];
    const config = node.configuration as Record<string, any>;
    if (config?.workspaceId) {
      metadata.push({ icon: "box", label: config.workspaceId });
    }
    if (config?.runId) {
      metadata.push({ icon: "play", label: config.runId });
    }

    return {
      iconSrc: terraformIcon,
      iconColor: getColorClass(componentDefinition.color),
      collapsedBackground: getBackgroundColorClass(componentDefinition.color),
      collapsed: node.isCollapsed,
      title: node.name || componentDefinition.label || componentDefinition.name || "Unnamed component",
      eventSections: lastExecution ? baseEventSections(nodes, lastExecution, componentName) : undefined,
      metadata,
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  subtitle(context: SubtitleContext): string {
    const { execution } = context;
    if (execution.result === "RESULT_FAILED") {
      return execution.resultMessage || "Execution failed";
    }
    return "Action completed successfully";
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const details: Record<string, string> = {};

    if (!outputs || !outputs.default || outputs.default.length === 0) {
      return details;
    }

    const outputData = outputs.default[0].data as Record<string, any>;
    
    // Map standard terraform run outputs
    if (outputData?.id) details["Run ID"] = outputData.id;
    if (outputData?.status) details["Status"] = outputData.status;
    if (outputData?.message) details["Message"] = outputData.message;
    if (outputData?.createdAt) details["Created At"] = outputData.createdAt;
    
    return details;
  },
};

function baseEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName!);
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent! });

  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventState: getState(componentName)(execution),
      eventSubtitle: execution.result === "RESULT_FAILED" ? execution.resultMessage || "Failed" : "Success",
      eventId: execution.rootEvent!.id!,
    },
  ];
}
