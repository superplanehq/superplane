import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import type { ComponentBaseProps, ComponentBaseSpec, EventSection } from "@/ui/componentBase";
import { getBackgroundColorClass, getColorClass } from "@/lib/colors";
import type React from "react";
import type { MetadataItem } from "@/ui/metadataList";
import { getTriggerRenderer } from "..";
import SemaphoreLogo from "@/assets/semaphore-logo-sign-black.svg";
import { renderTimeAgo } from "@/components/TimeAgo";
import { formatTimestampInUserTimezone } from "@/lib/timezone";
import { stringOrDash } from "../utils";
import { RUN_WORKFLOW_STATE_MAP, runWorkflowStateFunction } from "./run_workflow";

interface Configuration {
  project: string;
  pipelineId?: string;
  branch?: string;
  commitSha?: string;
  timeoutSeconds?: number;
  lookupTimeoutSeconds?: number;
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
  if (outputs.passed && outputs.passed.length > 0) {
    return outputs.passed[0].data as PipelineData;
  }

  if (outputs.failed && outputs.failed.length > 0) {
    return outputs.failed[0].data as PipelineData;
  }

  return undefined;
}

/**
 * Reuses the same state map/function as semaphore.runWorkflow: the execution
 * metadata shape (`{ workflow, pipeline: { state, result } }`) is identical.
 */
export {
  RUN_WORKFLOW_STATE_MAP as WAIT_FOR_PIPELINE_STATE_MAP,
  runWorkflowStateFunction as waitForPipelineStateFunction,
};

export const WAIT_FOR_PIPELINE_STATE_REGISTRY = {
  stateMap: RUN_WORKFLOW_STATE_MAP,
  getState: runWorkflowStateFunction,
};

export const waitForPipelineMapper: ComponentBaseMapper = {
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
      eventSections: lastExecution ? waitForPipelineEventSections(context.nodes, lastExecution) : undefined,
      includeEmptyState: !lastExecution,
      metadata: waitForPipelineMetadataList(context.node),
      specs: waitForPipelineSpecs(),
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
      if (metadata?.workflow) {
        return {
          "Workflow ID": metadata.workflow.id,
          "Workflow URL": metadata.workflow.url,
        };
      }

      return {};
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
    details["Repository URL"] = pipelineData.repository?.url;
    details["Workflow URL"] = pipelineData.workflow?.url;
    details["Finished At"] = pipelineData.pipeline?.done_at
      ? stringOrDash(formatTimestampInUserTimezone(pipelineData.pipeline.done_at))
      : "-";

    return details;
  },
};

function waitForPipelineMetadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as Configuration;
  const nodeMetadata = node.metadata as NodeMetadata;

  if (nodeMetadata?.project?.name) {
    metadata.push({ icon: "folder", label: nodeMetadata.project.name });
  } else if (configuration?.project) {
    metadata.push({ icon: "folder", label: configuration.project });
  }

  if (configuration?.pipelineId) {
    metadata.push({ icon: "hash", label: configuration.pipelineId });
    return metadata;
  }

  if (configuration?.branch) {
    metadata.push({ icon: "git-branch", label: configuration.branch });
  }

  if (configuration?.commitSha) {
    metadata.push({ icon: "git-commit", label: configuration.commitSha });
  }

  return metadata;
}

function waitForPipelineSpecs(): ComponentBaseSpec[] {
  return [];
}

function waitForPipelineEventSections(nodes: NodeInfo[], execution: ExecutionInfo): EventSection[] | undefined {
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
  if (revision?.commit_sha) {
    return `${revision.commit_sha.slice(0, 7)} · ${revision.commit_message}`;
  }

  return undefined;
}
