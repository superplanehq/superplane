import type React from "react";
import type { ComponentBaseProps } from "@/ui/componentBase";
import type { MetadataItem } from "@/ui/metadataList";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  OutputPayload,
  SubtitleContext,
} from "../types";
import { formatTimestamp } from "../utils";
import { baseProps } from "./base";
import type { CombinedCommitStatus, CommitStatus, GitLabNodeMetadata } from "./types";
import { buildGitlabExecutionSubtitle, shortSha } from "./utils";

interface GetCombinedCommitStatusConfiguration {
  project?: string;
  ref?: string;
}

export const getCombinedCommitStatusMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const props = baseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);
    const configuration = (context.node.configuration as GetCombinedCommitStatusConfiguration | undefined) ?? {};
    const metadata = (context.node.metadata as GitLabNodeMetadata | undefined) ?? ({} as GitLabNodeMetadata);

    const project = metadata?.project?.name || configuration.project;
    const metadataItems: MetadataItem[] = [];

    if (project) {
      metadataItems.push({ icon: "book", label: project });
    }

    if (configuration.ref) {
      metadataItems.push({ icon: "git-commit", label: shortSha(configuration.ref) });
    }

    return {
      ...props,
      metadata: metadataItems.length > 0 ? metadataItems : props.metadata,
    };
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const combined = outputs?.default?.[0]?.data as CombinedCommitStatus | undefined;
    return buildGitlabExecutionSubtitle(context.execution, combined?.state);
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const payload = outputs?.default?.[0];

    if (!payload?.data) {
      return {};
    }

    const combined = payload.data as CombinedCommitStatus;
    const statuses = combined.statuses || [];
    const counts = countStates(statuses);

    return {
      "Retrieved At": formatTimestamp(undefined, payload.timestamp),
      State: combined.state || "-",
      "Total statuses": (combined.total_count ?? statuses.length).toString(),
      Successful: counts.successful.toString(),
      Failed: counts.failed.toString(),
      SHA: combined.sha ? shortSha(combined.sha) : "-",
    };
  },
};

function countStates(statuses: CommitStatus[]): { successful: number; failed: number } {
  return statuses.reduce(
    (counts, status) => {
      if (status.status === "success") {
        counts.successful += 1;
      } else if (status.status === "failed" || status.status === "canceled") {
        counts.failed += 1;
      }
      return counts;
    },
    { successful: 0, failed: 0 },
  );
}
