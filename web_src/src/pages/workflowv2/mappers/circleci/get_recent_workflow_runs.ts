import { ComponentBaseProps } from "@/ui/componentBase";
import {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import { MetadataItem } from "@/ui/metadataList";
import { formatTimeAgo } from "@/utils/date";
import { stringOrDash } from "../utils";
import { baseProps } from "./base";

interface GetRecentWorkflowRunsConfiguration {
  projectSlug?: string;
  workflowName?: string;
  branch?: string;
}

interface WorkflowRunItem {
  id?: string;
  branch?: string;
  duration?: number;
  createdAt?: string;
  stoppedAt?: string;
  creditsUsed?: number;
  status?: string;
  isApproval?: boolean;
}

interface WorkflowRunsOutput {
  runs?: WorkflowRunItem[];
}

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as GetRecentWorkflowRunsConfiguration | undefined;

  if (configuration?.projectSlug) {
    metadata.push({ icon: "workflow", label: `Project: ${configuration.projectSlug}` });
  }

  if (configuration?.workflowName) {
    metadata.push({ icon: "play", label: `Workflow: ${configuration.workflowName}` });
  }

  return metadata;
}

function formatDuration(seconds?: number): string {
  if (seconds === undefined || seconds === null) {
    return "-";
  }
  if (seconds < 60) {
    return `${seconds}s`;
  }
  const minutes = Math.floor(seconds / 60);
  const remainingSeconds = seconds % 60;
  return remainingSeconds > 0 ? `${minutes}m ${remainingSeconds}s` : `${minutes}m`;
}

export const getRecentWorkflowRunsMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const base = baseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);
    return { ...base, metadata: metadataList(context.node) };
  },

  subtitle(context: SubtitleContext): string {
    const timestamp = context.execution.updatedAt || context.execution.createdAt;
    return timestamp ? formatTimeAgo(new Date(timestamp)) : "";
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const result = outputs?.default?.[0]?.data as WorkflowRunsOutput | undefined;
    const runs = result?.runs;

    const details: Record<string, string> = {
      "Retrieved At": context.execution.createdAt ? new Date(context.execution.createdAt).toLocaleString() : "-",
      "Total Runs": runs ? String(runs.length) : "-",
    };

    if (runs && runs.length > 0) {
      const latest = runs[0];
      details["Latest Run ID"] = stringOrDash(latest.id);
      details["Latest Status"] = stringOrDash(latest.status);
      details["Latest Branch"] = stringOrDash(latest.branch);
      details["Latest Duration"] = formatDuration(latest.duration);
      details["Latest Credits"] = stringOrDash(latest.creditsUsed);

      const succeeded = runs.filter((r) => r.status === "success").length;
      details["Success Rate"] = `${succeeded}/${runs.length} runs`;
    }

    return details;
  },
};
