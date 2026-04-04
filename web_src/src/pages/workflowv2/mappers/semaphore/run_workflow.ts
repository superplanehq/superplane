import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  EventStateRegistry,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  StateFunction,
  SubtitleContext,
} from "../types";
import type {
  ComponentBaseProps,
  ComponentBaseSpec,
  EventSection,
  EventState,
  EventStateMap,
} from "@/ui/componentBase";
import { DEFAULT_EVENT_STATE_MAP } from "@/ui/componentBase";
import { getBackgroundColorClass, getColorClass } from "@/lib/colors";
import type React from "react";
import type { MetadataItem } from "@/ui/metadataList";
import { getTriggerRenderer } from "..";
import SemaphoreLogo from "@/assets/semaphore-logo-sign-black.svg";
import { renderTimeAgo } from "@/components/TimeAgo";
import type { CanvasesCanvasNodeExecution } from "@/api-client";
import { formatTimestampInUserTimezone } from "@/lib/timezone";
import { stringOrDash } from "../utils";

interface Configuration {
  project: string;
  pipelineFile: string;
  ref: string;
  commitSha: string;
  parameters: Array<{ name: string; value: string }>;
}

interface NodeMetadata {
  project?: Project;
}

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

interface Outputs {
  passed?: OutputPayload[];
  failed?: OutputPayload[];
  default?: OutputPayload[];
}

interface PipelineData {
  project: Project;
  repository: Repository;
  revision: Revision;
  pipeline: Pipeline;
  workflow: Workflow;
  blocks: PipelineBlock[];
}

interface Project {
  id: string;
  name: string;
}

interface Workflow {
  id: string;
  url: string;
}

interface Pipeline {
  done_at: string;
  id: string;
  name: string;
  result: string;
  result_reason: string;
  state: string;
  working_directory: string;
  yaml_file_name: string;
}

interface Repository {
  slug: string;
  url: string;
}

interface Revision {
  branch: {
    commit_range: string;
    name: string;
  };
  commit_message: string;
  commit_sha: string;
  reference: string;
  reference_type: string;
}

interface PipelineBlock {
  name: string;
  jobs: Job[];
}

interface Job {
  id: string;
  index: number;
  name: string;
  result: string;
  status: string;
}

function getPipelineData(outputs: Outputs): PipelineData | undefined {
  return outputs?.passed?.[0].data ?? outputs?.failed?.[0].data ?? outputs?.default?.[0].data;
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
export const runWorkflowStateFunction: StateFunction = (execution: CanvasesCanvasNodeExecution): EventState => {
  if (!execution) return "neutral";

  if (
    execution.resultMessage &&
    (execution.resultReason === "RESULT_REASON_ERROR" ||
      (execution.result === "RESULT_FAILED" && execution.resultReason !== "RESULT_REASON_ERROR_RESOLVED"))
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
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;

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
      eventSections: lastExecution ? runWorkflowEventSections(context.nodes, lastExecution) : undefined,
      includeEmptyState: !lastExecution,
      metadata: runWorkflowMetadataList(context.node),
      specs: runWorkflowSpecs(context.node),
      eventStateMap: RUN_WORKFLOW_STATE_MAP,
    };
  },
  subtitle(context: SubtitleContext): string | React.ReactNode {
    const timestamp = context.execution.updatedAt || context.execution.createdAt;
    return timestamp ? renderTimeAgo(new Date(timestamp)) : "";
  },
  getExecutionDetails(context: ExecutionDetailsContext): Record<string, any> {
    //
    // If the execution is not finished, we just show
    // the information from the metadata.
    //
    if (context.execution.state !== "STATE_FINISHED") {
      const metadata = context.execution.metadata as ExecutionMetadata;
      return {
        "Workflow ID": metadata?.workflow?.id,
        "Workflow URL": metadata?.workflow?.url,
      };
    }

    //
    // If the execution is finished, we use the outputs to display more information.
    //
    const outputs = context.execution.outputs as Outputs;
    const pipelineData = getPipelineData(outputs);
    const details: Record<string, string> = {};
    if (!pipelineData) {
      return details;
    }

    if (pipelineData.project) {
      details["Project"] = pipelineData.project.name;
    }

    if (pipelineData.revision?.branch) {
      details["Branch"] = pipelineData.revision.branch.name;
    }

    details["Revision"] = stringOrDash(formatRevision(pipelineData.revision));
    details["Pipeline File"] = stringOrDash(formatPipelineFile(pipelineData.pipeline));
    details["Repository URL"] = pipelineData.repository?.url;
    details["Workflow URL"] = pipelineData.workflow?.url;
    details["Finished At"] = pipelineData.pipeline?.done_at
      ? stringOrDash(formatTimestampInUserTimezone(pipelineData.pipeline.done_at))
      : "-";

    return withBlockDetails(details, pipelineData.blocks || []);
  },
};

function runWorkflowMetadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as Configuration;
  const nodeMetadata = node.metadata as NodeMetadata;

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

function runWorkflowSpecs(node: NodeInfo): ComponentBaseSpec[] {
  const specs: ComponentBaseSpec[] = [];
  const configuration = node.configuration as Configuration;

  if (configuration?.parameters && configuration?.parameters.length > 0) {
    specs.push({
      title: "parameter",
      tooltipTitle: "workflow parameters",
      iconSlug: "settings",
      values: configuration?.parameters.map((param) => ({
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

function runWorkflowEventSections(nodes: NodeInfo[], execution: ExecutionInfo): EventSection[] | undefined {
  if (!execution) {
    return undefined;
  }

  const sections: EventSection[] = [];

  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName || "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });
  const executionState = runWorkflowStateFunction(execution);
  const subtitleTimestamp =
    executionState === "running" ? execution.createdAt : execution.updatedAt || execution.createdAt;
  const eventSubtitle = subtitleTimestamp ? renderTimeAgo(new Date(subtitleTimestamp)) : undefined;

  sections.push({
    receivedAt: new Date(execution.createdAt!),
    eventTitle: title,
    eventSubtitle,
    eventState: executionState,
    eventId: execution.rootEvent!.id!,
  });

  return sections;
}

function formatRevision(revision?: Revision): string | undefined {
  if (!revision) return undefined;
  return `${revision.commit_sha.slice(0, 7)} · ${revision.commit_message}`;
}

function formatPipelineFile(pipeline?: Pipeline): string | undefined {
  if (!pipeline) return undefined;
  return `${pipeline.working_directory}/${pipeline.yaml_file_name}`.replace("//", "/");
}

function withBlockDetails(details: Record<string, string>, blocks?: PipelineBlock[]): Record<string, string> {
  if (!blocks || blocks.length === 0) return details;

  for (const block of blocks) {
    const jobSummary = getJobSummaryForBlock(block);
    details[block.name] = `${jobSummary.passed} / ${jobSummary.total} jobs passed`;
  }

  return details;
}

interface JobCounts {
  total: number;
  passed: number;
  failed: number;
}

function getJobSummaryForBlock(block: PipelineBlock): JobCounts {
  const counts: JobCounts = { total: 0, passed: 0, failed: 0 };

  for (const job of block.jobs) {
    counts.total++;
    if (job.result === "passed") counts.passed++;
    if (job.result === "failed") counts.failed++;
  }

  return counts;
}
