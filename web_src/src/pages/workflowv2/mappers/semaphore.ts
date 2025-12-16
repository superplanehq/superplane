import {
  ComponentsComponent,
  ComponentsNode,
  WorkflowsWorkflowNodeExecution,
  WorkflowsWorkflowNodeQueueItem,
} from "@/api-client";
import { ComponentBaseMapper } from "./types";
import { ComponentBaseProps, ComponentBaseSpec, EventSection, EventState } from "@/ui/componentBase";
import { getColorClass } from "@/utils/colors";
import { MetadataItem } from "@/ui/metadataList";
import { getTriggerRenderer, getState, getStateMap } from ".";
import SemaphoreLogo from "@/assets/semaphore-logo-sign-black.svg";

export const semaphoreMapper: ComponentBaseMapper = {
  props(
    nodes: ComponentsNode[],
    node: ComponentsNode,
    componentDefinition: ComponentsComponent,
    lastExecutions: WorkflowsWorkflowNodeExecution[],
    nodeQueueItems?: WorkflowsWorkflowNodeQueueItem[],
  ): ComponentBaseProps {
    const componentName = componentDefinition.name || "semaphore";

    return {
      iconSrc: SemaphoreLogo,
      iconSlug: componentDefinition.icon || "workflow",
      headerColor: "bg-white",
      iconColor: getColorClass("black"),
      iconBackground: "bg-white",
      collapsed: node.isCollapsed,
      collapsedBackground: "bg-white",
      title: node.name!,
      eventSections: lastExecutions[0]
        ? getSemaphoreEventSections(nodes, lastExecutions[0], nodeQueueItems, componentName)
        : undefined,
      includeEmptyState: !lastExecutions[0],
      metadata: getSemaphoreMetadataList(node),
      specs: getSemaphoreSpecs(node),
      eventStateMap: getStateMap(componentName),
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

interface ExecutionMetadata {
  workflow?: {
    id: string;
    url: string;
    state: string;
    result: string;
  };
}

function getSemaphoreEventSections(
  nodes: ComponentsNode[],
  execution: WorkflowsWorkflowNodeExecution,
  _nodeQueueItems?: WorkflowsWorkflowNodeQueueItem[],
  componentName?: string,
): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.trigger?.name || "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle(execution.rootEvent!);

  const eventSection: EventSection = {
    showAutomaticTime: true,
    receivedAt: new Date(execution.createdAt!),
    eventTitle: title,
    eventState: componentName ? getState(componentName)(execution) : executionToEventSectionState(execution),
    eventId: execution.rootEvent?.id,
  };

  return [eventSection];
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
