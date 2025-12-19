import {
  ComponentsComponent,
  ComponentsNode,
  WorkflowsWorkflowNodeExecution,
  WorkflowsWorkflowNodeQueueItem,
} from "@/api-client";
import { ComponentBaseMapper, EventStateRegistry, StateFunction } from "./types";
import {
  ComponentBaseProps,
  ComponentBaseSpec,
  EventSection,
  EventState,
  EventStateMap,
  DEFAULT_EVENT_STATE_MAP,
} from "@/ui/componentBase";
import { getColorClass } from "@/utils/colors";
import { MetadataItem } from "@/ui/metadataList";
import { getTriggerRenderer } from ".";
import SemaphoreLogo from "@/assets/semaphore-logo-sign-black.svg";

interface ExecutionMetadata {
  workflow?: {
    id: string;
    url: string;
    state: string;
    result: string;
  };
}

export const SEMAPHORE_STATE_MAP: EventStateMap = {
  ...DEFAULT_EVENT_STATE_MAP,
  running: {
    icon: "loader-circle",
    textColor: "text-black",
    backgroundColor: "bg-blue-100",
    badgeColor: "bg-blue-500",
  },
  passed: {
    icon: "circle-check",
    textColor: "text-black",
    backgroundColor: "bg-green-100",
    badgeColor: "bg-emerald-500",
  },
  failed: {
    icon: "circle-x",
    textColor: "text-black",
    backgroundColor: "bg-red-100",
    badgeColor: "bg-red-400",
  },
  stopped: {
    icon: "circle-stop",
    textColor: "text-black",
    backgroundColor: "bg-gray-100",
    badgeColor: "bg-gray-500",
  },
};

/**
 * Semaphore-specific state logic function
 */
export const semaphoreStateFunction: StateFunction = (execution: WorkflowsWorkflowNodeExecution): EventState => {
  if (!execution) return "neutral";

  // Check if workflow is still running
  if (execution.state === "STATE_PENDING" || execution.state === "STATE_STARTED") {
    return "running";
  }

  // Check workflow result from metadata
  const metadata = execution.metadata as ExecutionMetadata;
  if (execution.state === "STATE_FINISHED") {
    if (execution.result === "RESULT_PASSED") {
      if (metadata.workflow?.result === "passed") {
        return "passed";
      }
      if (metadata.workflow?.result === "failed") {
        return "failed";
      }
      if (metadata.workflow?.result === "stopped") {
        return "stopped";
      }
      return "passed";
    }
    if (execution.result === "RESULT_FAILED") {
      return "failed";
    }
  }

  return "failed";
};

/**
 * Semaphore-specific state registry
 */
export const SEMAPHORE_STATE_REGISTRY: EventStateRegistry = {
  stateMap: SEMAPHORE_STATE_MAP,
  getState: semaphoreStateFunction,
};

export const semaphoreMapper: ComponentBaseMapper = {
  props(
    nodes: ComponentsNode[],
    node: ComponentsNode,
    componentDefinition: ComponentsComponent,
    lastExecutions: WorkflowsWorkflowNodeExecution[],
    nodeQueueItems?: WorkflowsWorkflowNodeQueueItem[],
  ): ComponentBaseProps {
    return {
      iconSrc: SemaphoreLogo,
      iconSlug: componentDefinition.icon || "workflow",
      headerColor: "bg-white",
      iconColor: getColorClass("black"),
      iconBackground: "bg-white",
      collapsed: node.isCollapsed,
      collapsedBackground: "bg-white",
      title: node.name!,
      eventSections: lastExecutions[0] ? getSemaphoreEventSections(nodes, lastExecutions[0], nodeQueueItems) : undefined,
      includeEmptyState: !lastExecutions[0],
      metadata: getSemaphoreMetadataList(node),
      specs: getSemaphoreSpecs(node),
      eventStateMap: SEMAPHORE_STATE_MAP,
    };
  },
};

function getSemaphoreMetadataList(node: ComponentsNode): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as Record<string, unknown>;
  const nodeMetadata = node.metadata as Record<string, unknown>;

  if ((nodeMetadata as any)?.project?.name) {
    metadata.push({ icon: "folder", label: (nodeMetadata as any).project.name });
  } else if (configuration?.project) {
    metadata.push({ icon: "folder", label: configuration.project as string });
  }

  if (configuration?.ref) {
    metadata.push({ icon: "git-branch", label: configuration.ref as string });
  }

  if (configuration?.pipelineFile) {
    metadata.push({ icon: "file-code", label: configuration.pipelineFile as string });
  }

  if (configuration?.commitSha) {
    metadata.push({ icon: "git-commit", label: configuration.commitSha as string });
  }

  return metadata;
}

function getSemaphoreSpecs(node: ComponentsNode): ComponentBaseSpec[] {
  const specs: ComponentBaseSpec[] = [];
  const configuration = node.configuration as Record<string, unknown>;

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

function getSemaphoreEventSections(
  nodes: ComponentsNode[],
  execution: WorkflowsWorkflowNodeExecution,
  _nodeQueueItems?: WorkflowsWorkflowNodeQueueItem[],
): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.trigger?.name || "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle(execution.rootEvent!);

  const eventSection: EventSection = {
    showAutomaticTime: true,
    receivedAt: new Date(execution.createdAt!),
    eventTitle: title,
    eventState: semaphoreStateFunction(execution),
    eventId: execution.rootEvent?.id,
  };

  return [eventSection];
}
