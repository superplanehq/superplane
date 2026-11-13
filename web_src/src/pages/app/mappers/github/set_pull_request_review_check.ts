import type { ComponentBaseMapper, ExecutionDetailsContext, OutputPayload } from "../types";
import { baseProps } from "./base";
import { publishCommitStatusMapper } from "./publish_commit_status";

interface ReviewCheckConfiguration {
  repository?: string;
  pullNumber?: string | number;
  context?: string;
  phase?: string;
  reviewers?: { id: number; login: string }[];
  reason?: string;
  resultUrl?: string;
}

interface ReviewCheckOutput {
  status?: string;
  state?: string;
  sha?: string;
  phase?: string;
  context?: string;
  generation?: number;
  activationGeneration?: number;
}

export const setPullRequestReviewCheckMapper: ComponentBaseMapper = {
  props(context) {
    return baseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);
  },
  subtitle: publishCommitStatusMapper.subtitle,
  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const configuration = (context.execution.configuration ||
      context.node.configuration ||
      {}) as ReviewCheckConfiguration;
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const output = outputs?.default?.[0]?.data as ReviewCheckOutput | undefined;
    return { ...configurationDetails(configuration), ...outputDetails(output) };
  },
};

function configurationDetails(configuration: ReviewCheckConfiguration): Record<string, string> {
  return {
    Repository: configuration.repository || "-",
    "Pull request": configuration.pullNumber ? `#${configuration.pullNumber}` : "-",
    Context: configuration.context || "-",
    Phase: (configuration.phase || "-").replace(/_/g, " "),
    Reviewers: configuration.reviewers?.map((reviewer) => reviewer.login).join(", ") || "-",
    Reason: configuration.reason || "-",
    "Result URL": configuration.resultUrl || "-",
  };
}

function outputDetails(output: ReviewCheckOutput | undefined): Record<string, string> {
  if (!output) return {};
  return {
    Status: output.status || "-",
    "Commit status": output.state || "-",
    Commit: output.sha || "-",
    Generation: output.generation?.toString() || "-",
    "Activation generation": output.activationGeneration?.toString() || "-",
  };
}
