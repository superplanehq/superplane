import type React from "react";
import type { ComponentBaseProps } from "@/ui/componentBase";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  OutputPayload,
  SubtitleContext,
} from "../types";
import { stringOrDash } from "../utils";
import { baseProps } from "./base";
import { buildGithubExecutionSubtitle } from "./utils";

interface CommitStatus {
  state?: string;
  context?: string;
  description?: string;
  target_url?: string;
}

interface CombinedCommitStatus {
  state?: string;
  sha?: string;
  total_count?: number;
  statuses?: CommitStatus[];
  commit_url?: string;
  repository_url?: string;
}

interface StatusCounts {
  successful: number;
  failed: number;
  errored: number;
  pending: number;
}

export const getCombinedCommitStatusMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return baseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    return buildGithubExecutionSubtitle(context.execution);
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const status = combinedCommitStatusOutput(context);
    if (!status) {
      return {};
    }

    const statuses = status.statuses || [];
    const counts = countStatusStates(statuses);

    return {
      State: stringOrDash(status.state),
      "Total statuses": statusTotal(status, statuses),
      Successful: counts.successful.toString(),
      Failed: counts.failed.toString(),
      Errored: counts.errored.toString(),
      Pending: counts.pending.toString(),
      "First non-success status": stringOrDash(firstNonSuccessContext(statuses)),
      SHA: stringOrDash(shortSha(status.sha)),
      "Commit URL": stringOrDash(status.commit_url),
      "Repository URL": stringOrDash(status.repository_url),
    };
  },
};

function combinedCommitStatusOutput(context: ExecutionDetailsContext): CombinedCommitStatus | undefined {
  const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
  const firstOutput = outputs?.default?.[0];
  return firstOutput?.data as CombinedCommitStatus | undefined;
}

function countStatusStates(statuses: CommitStatus[]): StatusCounts {
  return statuses.reduce(
    (counts, status) => {
      switch (status.state) {
        case "success":
          counts.successful += 1;
          break;
        case "failure":
          counts.failed += 1;
          break;
        case "error":
          counts.errored += 1;
          break;
        case "pending":
          counts.pending += 1;
          break;
      }

      return counts;
    },
    { successful: 0, failed: 0, errored: 0, pending: 0 },
  );
}

function firstNonSuccessContext(statuses: CommitStatus[]): string | undefined {
  const status = statuses.find((item) => item.state !== "success");
  return status?.context || status?.description || status?.target_url;
}

function statusTotal(status: CombinedCommitStatus, statuses: CommitStatus[]): string {
  return (status.total_count ?? statuses.length).toString();
}

function shortSha(sha: string | undefined): string | undefined {
  return sha ? sha.slice(0, 7) : undefined;
}
