import {
  ComponentsComponent,
  ComponentsNode,
  WorkflowsWorkflowNodeExecution,
  WorkflowsWorkflowNodeQueueItem,
} from "@/api-client";
import { ComponentBaseMapper } from "../types";
import { ComponentBaseProps, ComponentBaseSpec, EventSection, EventState } from "@/ui/componentBase";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { MetadataItem } from "@/ui/metadataList";
import { getTriggerRenderer, getState, getStateMap } from "..";
import SemaphoreLogo from "@/assets/semaphore-logo-sign-black.svg";

export const runWorkflowMapper: ComponentBaseMapper = {
  props(
    nodes: ComponentsNode[],
    node: ComponentsNode,
    componentDefinition: ComponentsComponent,
    lastExecutions: WorkflowsWorkflowNodeExecution[],
    nodeQueueItems?: WorkflowsWorkflowNodeQueueItem[],
  ): ComponentBaseProps {
    const componentName = componentDefinition.name || "semaphore";

    return {
      title: node.name!,
      iconSrc: SemaphoreLogo,
      iconSlug: componentDefinition.icon || "workflow",
      headerColor: getBackgroundColorClass(componentDefinition?.color || "gray"),
      iconColor: getColorClass(componentDefinition?.color || "gray"),
      iconBackground: getBackgroundColorClass(componentDefinition?.color || "gray"),
      collapsed: node.isCollapsed,
      collapsedBackground: getBackgroundColorClass("white"),
      eventSections: runWorkflowEventSections(nodes, lastExecutions[0], nodeQueueItems, componentName),
      includeEmptyState: !hasExecutionOrQueueItems(lastExecutions[0], nodeQueueItems),
      metadata: runWorkflowMetadataList(node),
      specs: runWorkflowSpecs(node),
      eventStateMap: getStateMap(componentName),
    };
  },
};

function runWorkflowMetadataList(node: ComponentsNode): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as any;
  const nodeMetadata = node.metadata as any;

  if (nodeMetadata?.project?.name) {
    metadata.push({ icon: "folder", label: nodeMetadata.project.name });
  } else if (configuration?.project) {
    metadata.push({ icon: "folder", label: configuration.project });
  }

  if (configuration?.ref) {
    metadata.push({ icon: "git-branch", label: configuration.ref });
  }

  if (configuration?.pipelineFile) {
    metadata.push({ icon: "file-code", label: configuration.pipelineFile });
  }

  if (configuration?.commitSha) {
    metadata.push({ icon: "git-commit", label: configuration.commitSha });
  }

  return metadata;
}

function runWorkflowSpecs(node: ComponentsNode): ComponentBaseSpec[] {
  const specs: ComponentBaseSpec[] = [];
  const configuration = node.configuration as any;

  const parameters = configuration?.parameters as Array<{ name: string; value: string }> | undefined;
  if (parameters && parameters.length > 0) {
    specs.push({
      title: "parameter",
      tooltipTitle: "workflow parameters",
      iconSlug: "settings",
      values: parameters.map((param) => ({
        badges: [
          {
            label: param.name,
            bgColor: "bg-purple-100",
            textColor: "text-purple-800",
          },
          {
            label: param.value,
            bgColor: "bg-gray-100",
            textColor: "text-gray-800",
          },
        ],
      })),
    });
  }

  return specs;
}

interface ExecutionMetadata {
  workflow?: {
    id: string;
    url: string;
    state: string;
    result: string;
  };
}

function hasExecutionOrQueueItems(
  execution: WorkflowsWorkflowNodeExecution,
  nodeQueueItems?: WorkflowsWorkflowNodeQueueItem[],
): boolean {
  return !!execution || (nodeQueueItems && nodeQueueItems.length > 0);
}

function runWorkflowEventSections(
  nodes: ComponentsNode[],
  execution: WorkflowsWorkflowNodeExecution,
  nodeQueueItems?: WorkflowsWorkflowNodeQueueItem[],
  componentName?: string,
): EventSection[] | undefined {
  if (!hasExecutionOrQueueItems(execution, nodeQueueItems)) {
    return undefined;
  }

  const sections: EventSection[] = [];

  //
  // If there is an execution, add section for execution.
  //
  if (execution) {
    const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
    const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.trigger?.name || "");
    const { title } = rootTriggerRenderer.getTitleAndSubtitle(execution.rootEvent!);
    sections.push({
      showAutomaticTime: true,
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventState: componentName ? getState(componentName)(execution) : executionToEventSectionState(execution),
    });
  }

  //
  // If there are queue items, add section for next in queue.
  //
  if (nodeQueueItems && nodeQueueItems.length > 0) {
    const queueItem = nodeQueueItems[nodeQueueItems.length - 1];
    const rootTriggerNode = nodes.find((n) => n.id === queueItem.rootEvent?.nodeId);
    const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.trigger?.name || "");

    if (queueItem.rootEvent) {
      const { title } = rootTriggerRenderer.getTitleAndSubtitle(queueItem.rootEvent);
      sections.push({
        receivedAt: queueItem.createdAt ? new Date(queueItem.createdAt) : undefined,
        eventTitle: title,
        eventState: "next-in-queue" as const,
      });
    }
  }

  return sections;
}

function executionToEventSectionState(execution: WorkflowsWorkflowNodeExecution): EventState {
  if (execution.state == "STATE_PENDING" || execution.state == "STATE_STARTED") {
    return "running";
  }

  const metadata = execution.metadata as ExecutionMetadata;
  if (metadata.workflow?.result === "passed") {
    return "success";
  }

  return "failed";
}
