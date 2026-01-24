import {
  ComponentsComponent,
  ComponentsNode,
  WorkflowsWorkflowNodeExecution,
  WorkflowsWorkflowNodeQueueItem,
} from "@/api-client";
import { ComponentBaseMapper, EventStateRegistry, OutputPayload, StateFunction } from "../types";
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
import { formatTimeAgo } from "@/utils/date";

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
  const pipelineResult = metadata?.pipeline?.result;
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
      appName: "semaphore",
      iconSlug: componentDefinition.icon || "workflow",
      iconColor: getColorClass(componentDefinition?.color || "gray"),
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
    const timestamp = execution.updatedAt || execution.createdAt;
    return timestamp ? formatTimeAgo(new Date(timestamp)) : "";
  },
  getExecutionDetails(execution: WorkflowsWorkflowNodeExecution, _node: ComponentsNode): Record<string, any> {
    const details: Record<string, any> = {};
    const outputs = execution.outputs as
      | { passed?: OutputPayload[]; failed?: OutputPayload[]; default?: OutputPayload[] }
      | undefined;
    const payload =
      (outputs?.passed?.[0]?.data as Record<string, any> | undefined) ||
      (outputs?.failed?.[0]?.data as Record<string, any> | undefined) ||
      (outputs?.default?.[0]?.data as Record<string, any> | undefined);
    const payloadData =
      payload && typeof payload === "object" && payload.data && typeof payload.data === "object"
        ? payload.data
        : payload;
    const metadataFallback =
      (!payloadData || typeof payloadData !== "object") && execution.metadata
        ? (execution.metadata as Record<string, any>)
        : undefined;

    const sourceData =
      payloadData && typeof payloadData === "object"
        ? payloadData
        : metadataFallback && typeof metadataFallback === "object"
          ? metadataFallback
          : undefined;

    if (!sourceData || typeof sourceData !== "object") {
      return details;
    }

    const pipeline = sourceData.pipeline as Record<string, any> | undefined;
    const repository = sourceData.repository as Record<string, any> | undefined;
    const project = sourceData.project as Record<string, any> | undefined;
    const organization = sourceData.organization as Record<string, any> | undefined;
    const revision = sourceData.revision as Record<string, any> | undefined;
    const blocks = sourceData.blocks as Array<Record<string, any>> | undefined;
    const workflow = sourceData.workflow as Record<string, any> | undefined;

    const addDetail = (key: string, value?: string) => {
      if (value) {
        details[key] = value;
      }
    };

    addDetail("Done At", formatDate(pipeline?.done_at));
    addDetail("Workflow URL", (execution.metadata as Record<string, any> | undefined)?.workflow?.url || workflow?.url);
    addDetail("Repository URL", repository?.url);
    addDetail("Project", project?.name);
    addDetail("Organization", organization?.name);
    addDetail("Branch", revision?.branch?.name || revision?.reference);
    addDetail("Commit", formatCommit(revision));
    addDetail("Pipeline File", formatPipelineFile(pipeline));
    const blockDetails = buildBlocksDetails(blocks);
    if (blockDetails) {
      details["Blocks"] = blockDetails;
    }

    return details;
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
    const executionState = runWorkflowStateFunction(execution);
    const subtitleTimestamp =
      executionState === "running" ? execution.createdAt : execution.updatedAt || execution.createdAt;
    const eventSubtitle = subtitleTimestamp ? formatTimeAgo(new Date(subtitleTimestamp)) : undefined;

    sections.push({
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventSubtitle,
      eventState: executionState,
      eventId: execution.rootEvent?.id,
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
        eventSubtitle: queueItem.createdAt ? formatTimeAgo(new Date(queueItem.createdAt)) : undefined,
        eventState: "next-in-queue" as const,
        eventId: queueItem.rootEvent?.id,
      });
    }
  }

  return sections;
}

function formatDate(value?: string): string | undefined {
  if (!value) return undefined;
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return undefined;
  return date.toLocaleString();
}

function formatCommit(revision?: Record<string, any>): string | undefined {
  if (!revision) return undefined;
  const sha = revision.commit_sha as string | undefined;
  const message = revision.commit_message as string | undefined;
  const shortSha = sha ? sha.slice(0, 7) : undefined;
  if (shortSha && message) return `${shortSha} · ${message}`;
  return shortSha || message;
}

function formatPipelineFile(pipeline?: Record<string, any>): string | undefined {
  if (!pipeline) return undefined;
  const workingDirectory = pipeline.working_directory as string | undefined;
  const yamlFileName = pipeline.yaml_file_name as string | undefined;
  if (workingDirectory && yamlFileName) return `${workingDirectory}/${yamlFileName}`.replace("//", "/");
  return yamlFileName || workingDirectory;
}

function buildBlocksDetails(blocks?: Array<Record<string, any>>): Record<string, any> | undefined {
  if (!blocks || blocks.length === 0) return undefined;

  return {
    __type: "semaphoreBlocks",
    blocks: blocks.map((block) => {
      const jobs = (block?.jobs as Array<Record<string, any>> | undefined) || [];
      return {
        name: block?.name as string | undefined,
        result: block?.result as string | undefined,
        resultReason: block?.result_reason as string | undefined,
        state: block?.state as string | undefined,
        jobs: jobs.map((job) => ({
          name: job?.name as string | undefined,
          result: job?.result as string | undefined,
          status: job?.status as string | undefined,
        })),
      };
    }),
  };
}
