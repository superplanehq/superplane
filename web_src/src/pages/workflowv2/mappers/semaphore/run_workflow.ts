import {
  ComponentsComponent,
  ComponentsNode,
  WorkflowsWorkflowNodeExecution,
  WorkflowsWorkflowNodeQueueItem,
} from "@/api-client";
import { ComponentBaseMapper, EventStateRegistry, StateFunction } from "../types";
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
import { getTriggerRenderer } from "..";
import SemaphoreLogo from "@/assets/semaphore-logo-sign-black.svg";

interface ExecutionMetadata {
  workflow?: {
    id: string;
    url: string;
  };
  pipeline?: {
    state: string;
    result: string;
  };
}

export const RUN_WORKFLOW_STATE_MAP: EventStateMap = {
  ...DEFAULT_EVENT_STATE_MAP,
  running: {
    icon: "loader-circle",
    textColor: "text-gray-800",
    backgroundColor: "bg-blue-100",
    badgeColor: "bg-blue-500",
  },
  passed: {
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
  stopped: {
    icon: "circle-stop",
    textColor: "text-gray-800",
    backgroundColor: "bg-gray-100",
    badgeColor: "bg-gray-500",
  },
};

/**
 * Semaphore-specific state logic function
 */
export const runWorkflowStateFunction: StateFunction = (execution: WorkflowsWorkflowNodeExecution): EventState => {
  if (!execution) return "neutral";

  if (
    execution.resultMessage &&
    (execution.resultReason === "RESULT_REASON_ERROR" || execution.result === "RESULT_FAILED")
  ) {
    return "error";
  }

  if (execution.result === "RESULT_CANCELLED") {
    return "cancelled";
  }

  //
  // If workflow is still running
  //
  if (execution.state === "STATE_PENDING" || execution.state === "STATE_STARTED") {
    return "running";
  }

  const metadata = execution.metadata as ExecutionMetadata;
  const pipelineResult = metadata.pipeline?.result;
  if (pipelineResult === "failed") {
    return "failed";
  }
  if (pipelineResult === "stopped") {
    return "stopped";
  }

  return "passed";
};

/**
 * Semaphore-specific state registry
 */
export const RUN_WORKFLOW_STATE_REGISTRY: EventStateRegistry = {
  stateMap: RUN_WORKFLOW_STATE_MAP,
  getState: runWorkflowStateFunction,
};

export const runWorkflowMapper: ComponentBaseMapper = {
  props(
    nodes: ComponentsNode[],
    node: ComponentsNode,
    componentDefinition: ComponentsComponent,
    lastExecutions: WorkflowsWorkflowNodeExecution[],
    nodeQueueItems?: WorkflowsWorkflowNodeQueueItem[],
  ): ComponentBaseProps {
    return {
      title: node.name!,
      iconSrc: SemaphoreLogo,
      iconSlug: componentDefinition.icon || "workflow",
      headerColor: getBackgroundColorClass(componentDefinition?.color || "gray"),
      iconColor: getColorClass(componentDefinition?.color || "gray"),
      iconBackground: getBackgroundColorClass(componentDefinition?.color || "gray"),
      collapsed: node.isCollapsed,
      collapsedBackground: getBackgroundColorClass("white"),
      eventSections: runWorkflowEventSections(nodes, lastExecutions[0], nodeQueueItems),
      includeEmptyState: !hasExecutionOrQueueItems(lastExecutions[0], nodeQueueItems),
      metadata: runWorkflowMetadataList(node),
      specs: runWorkflowSpecs(node),
      eventStateMap: RUN_WORKFLOW_STATE_MAP,
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
      eventState: runWorkflowStateFunction(execution),
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
