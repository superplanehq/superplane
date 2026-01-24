import React, { useState } from "react";
import {
  ComponentsComponent,
  ComponentsNode,
  WorkflowsWorkflowNodeExecution,
  WorkflowsWorkflowNodeQueueItem,
} from "@/api-client";
import { ComponentBaseMapper, CustomFieldRenderer, EventStateRegistry, StateFunction } from "../types";
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
import { buildGithubExecutionSubtitle } from "./utils";
import { Icon } from "@/components/Icon";

interface ExecutionMetadata {
  workflowRun?: {
    id: number;
    status: string;
    conclusion: string;
    url: string;
  };
}

export const RUN_WORKFLOW_STATE_MAP: EventStateMap = {
  ...DEFAULT_EVENT_STATE_MAP,
  running: {
    icon: "loader-circle",
    textColor: "text-gray-800 dark:text-gray-200",
    backgroundColor: "bg-blue-100 dark:bg-blue-900/50",
    badgeColor: "bg-blue-500",
  },
  passed: {
    icon: "circle-check",
    textColor: "text-gray-800 dark:text-gray-200",
    backgroundColor: "bg-green-100 dark:bg-green-900/50",
    badgeColor: "bg-emerald-500",
  },
  failed: {
    icon: "circle-x",
    textColor: "text-gray-800 dark:text-gray-200",
    backgroundColor: "bg-red-100 dark:bg-red-900/50",
    badgeColor: "bg-red-400",
  },
  stopped: {
    icon: "circle-stop",
    textColor: "text-gray-800 dark:text-gray-200",
    backgroundColor: "bg-gray-100 dark:bg-gray-700",
    badgeColor: "bg-gray-500",
  },
};

/**
 * GitHub-specific state logic function
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
  switch (metadata.workflowRun?.conclusion) {
    case "cancelled":
      return "stopped";

    case "failure":
      return "failed";

    default:
      return "passed";
  }
};

/**
 * GitHub-specific run workflow state registry
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
      appName: "github",
      iconColor: getColorClass(componentDefinition?.color!),
      collapsed: node.isCollapsed,
      collapsedBackground: getBackgroundColorClass("white"),
      eventSections: runWorkflowEventSections(nodes, lastExecutions[0], nodeQueueItems),
      includeEmptyState: !hasExecutionOrQueueItems(lastExecutions[0], nodeQueueItems),
      metadata: runWorkflowMetadataList(node),
      specs: runWorkflowSpecs(node),
      eventStateMap: RUN_WORKFLOW_STATE_MAP,
    };
  },
  subtitle(_node: ComponentsNode, execution: WorkflowsWorkflowNodeExecution): string {
    return buildGithubExecutionSubtitle(execution);
  },

  getExecutionDetails(execution: WorkflowsWorkflowNodeExecution, _node: ComponentsNode): Record<string, string> {
    const metadata = execution.metadata as ExecutionMetadata;
    const details: Record<string, string> = {};

    if (metadata.workflowRun?.url) {
      details["Workflow URL"] = metadata.workflowRun.url;
    }

    if (metadata.workflowRun?.id) {
      details["Run ID"] = metadata.workflowRun.id.toString();
    }

    if (metadata.workflowRun?.status) {
      details["Run Status"] = metadata.workflowRun.status;
    }

    if (metadata.workflowRun?.conclusion) {
      details["Conclusion"] = metadata.workflowRun.conclusion;
    }

    return details;
  },
};

function runWorkflowMetadataList(node: ComponentsNode): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as any;
  const nodeMetadata = node.metadata as any;

  if (nodeMetadata?.repository?.name) {
    metadata.push({ icon: "book", label: nodeMetadata.repository.name });
  }

  if (configuration?.ref) {
    metadata.push({ icon: "git-branch", label: configuration.ref });
  }

  if (configuration?.workflowFile) {
    metadata.push({ icon: "file-code", label: configuration.workflowFile });
  }

  return metadata;
}

function runWorkflowSpecs(node: ComponentsNode): ComponentBaseSpec[] {
  const specs: ComponentBaseSpec[] = [];
  const configuration = node.configuration as any;

  const inputs = configuration?.inputs as Array<{ name: string; value: string }> | undefined;
  if (inputs && inputs.length > 0) {
    specs.push({
      title: "input",
      tooltipTitle: "inputs",
      iconSlug: "settings",
      values: inputs.map((param) => ({
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

/**
 * Copy button component for code blocks
 */
const CopyCodeButton: React.FC<{ code: string }> = ({ code }) => {
  const [copied, setCopied] = useState(false);

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(code);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch (err) {
      console.error("Failed to copy text: ", err);
    }
  };

  return (
    <button
      onClick={handleCopy}
      className="absolute top-2 right-2 z-10 opacity-0 group-hover:opacity-100 transition-opacity p-1 bg-white outline-1 outline-black/20 hover:outline-black/30 rounded text-gray-600 dark:text-gray-400"
      title={copied ? "Copied!" : "Copy to clipboard"}
    >
      <Icon name={copied ? "check" : "copy"} size="sm" />
    </button>
  );
};

/**
 * Generate the workflow YAML snippet based on user's inputs
 */
function generateWorkflowYamlSnippet(userInputs: Array<{ name: string; value: string }> | undefined): string {
  let inputsSection = `      superplane_canvas_id:
        required: true
        type: string
      superplane_execution_id:
        required: true
        type: string`;

  // Add user-defined inputs (not required)
  if (userInputs && userInputs.length > 0) {
    for (const input of userInputs) {
      if (input.name && input.name.trim()) {
        inputsSection += `
      ${input.name}:
        required: false
        type: string`;
      }
    }
  }

  return `# Controls when the workflow will run
on:
  workflow_dispatch:
    inputs:
${inputsSection}

run-name: "My workflow - \${{ inputs.superplane_execution_id }}"`;
}

/**
 * Custom field renderer for GitHub Run Workflow component configuration
 */
export const runWorkflowCustomFieldRenderer: CustomFieldRenderer = {
  render: (_node: ComponentsNode, configuration: Record<string, unknown>) => {
    const inputs = configuration?.inputs as Array<{ name: string; value: string }> | undefined;
    const yamlSnippet = generateWorkflowYamlSnippet(inputs);

    return (
      <div className="border-t-1 border-gray-200 pt-4">
        <div className="space-y-3">
          <div>
            <span className="text-sm font-medium text-gray-700 dark:text-gray-300">Workflow Configuration</span>
            <div className="text-xs text-gray-800 dark:text-gray-100 mt-2 border-1 border-orange-950/20 px-2.5 py-2 bg-orange-50 dark:bg-amber-800 rounded-md">
              <p className="mb-3">
                In order for SuperPlane to track GitHub Workflow execution, you need to add two inputs to your workflow
                and include one of them in the run name.
              </p>
              <div className="relative group">
                <pre className="text-xs text-gray-800 dark:text-gray-100 border-1 border-gray-300 dark:border-gray-600 px-2.5 py-2 bg-gray-50 dark:bg-gray-800 rounded-md font-mono whitespace-pre overflow-x-auto">
                  {yamlSnippet}
                </pre>
                <CopyCodeButton code={yamlSnippet} />
              </div>
            </div>
          </div>
        </div>
      </div>
    );
  },
};
