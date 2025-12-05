import {
  ComponentsComponent,
  ComponentsNode,
  WorkflowsWorkflowNodeExecution,
  WorkflowsWorkflowNodeQueueItem,
} from "@/api-client";
import { ComponentBaseMapper } from "./types";
import { ComponentBaseProps, ComponentBaseSpec, EventSection } from "@/ui/componentBase";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { MetadataItem } from "@/ui/metadataList";
import { getTriggerRenderer } from ".";
import SemaphoreLogo from "@/assets/semaphore-logo-sign-black.svg";
import { success, failed, neutral, running, inQueue } from "./eventSectionUtils";

export const semaphoreMapper: ComponentBaseMapper = {
  props(
    nodes: ComponentsNode[],
    node: ComponentsNode,
    componentDefinition: ComponentsComponent,
    lastExecution: WorkflowsWorkflowNodeExecution,
    nodeQueueItems?: WorkflowsWorkflowNodeQueueItem[],
  ): ComponentBaseProps {
    return {
      iconSrc: SemaphoreLogo,
      iconSlug: componentDefinition.icon || "workflow",
      headerColor: getBackgroundColorClass(componentDefinition?.color || "gray"),
      iconColor: getColorClass(componentDefinition?.color || "gray"),
      iconBackground: getBackgroundColorClass(componentDefinition?.color || "gray"),
      collapsed: node.isCollapsed,
      collapsedBackground: getBackgroundColorClass("white"),
      title: node.name!,
      eventSections: getSemaphoreEventSections(nodes, lastExecution, nodeQueueItems),
      metadata: getSemaphoreMetadataList(node),
      specs: getSemaphoreSpecs(node),
    };
  },
};

function getSemaphoreMetadataList(node: ComponentsNode): MetadataItem[] {
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

function getSemaphoreSpecs(node: ComponentsNode): ComponentBaseSpec[] {
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
  url: string;
  workflow?: {
    id: string;
    pipeline?: {
      state: string;
      result: string;
    }
  };
}

function getSemaphoreEventSections(
  nodes: ComponentsNode[],
  execution: WorkflowsWorkflowNodeExecution,
  nodeQueueItems?: WorkflowsWorkflowNodeQueueItem[],
): EventSection[] {
  const sections: EventSection[] = [];

  // Add Last Run section
  if (!execution) {
    sections.push(
      neutral({
        title: "Last Run",
        eventTitle: "No executions received yet",
      }),
    );
  } else {
    const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
    const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.trigger?.name || "");
    const { title } = rootTriggerRenderer.getTitleAndSubtitle(execution.rootEvent!);
    sections.push(lastRunSection(execution, title));
  }

  // Add Next in Queue section if there are queued items
  if (nodeQueueItems && nodeQueueItems.length > 0) {
    const queueItem = nodeQueueItems[nodeQueueItems.length - 1];
    const rootTriggerNode = nodes.find((n) => n.id === queueItem.rootEvent?.nodeId);
    const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.trigger?.name || "");

    if (queueItem.rootEvent) {
      const { title } = rootTriggerRenderer.getTitleAndSubtitle(queueItem.rootEvent);
      sections.push(
        inQueue({
          title: "Next in Queue",
          receivedAt: queueItem.createdAt ? new Date(queueItem.createdAt) : undefined,
          eventTitle: title,
        }),
      );
    }
  }

  return sections;
}

function lastRunSection(execution: WorkflowsWorkflowNodeExecution, title: string): EventSection {
  const baseProps = {
    title: "Last Run",
    showAutomaticTime: true,
    receivedAt: new Date(execution.createdAt!),
    eventTitle: title,
  };

  if (execution.state == "STATE_PENDING" || execution.state == "STATE_STARTED") {
    return running(baseProps);
  }

  const metadata = execution.metadata as unknown as ExecutionMetadata;
  const result = metadata.workflow?.pipeline?.result;

  switch (result) {
    case "passed":
      return success(baseProps);
    case "stopped":
      return {
        ...baseProps,
        iconSlug: "circle-stop",
        textColor: "text-gray-600",
        backgroundColor: "bg-gray-200",
        iconColor: "text-gray-500 bg-gray-500",
        iconSize: 12,
        iconClassName: "text-white",
        inProgress: false,
      };
    default:
      return failed(baseProps);
  }
}
