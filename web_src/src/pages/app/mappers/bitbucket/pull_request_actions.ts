import type React from "react";
import type { ComponentBaseProps } from "@/ui/componentBase";
import type { ComponentBaseContext, ComponentBaseMapper, ExecutionDetailsContext, SubtitleContext } from "../types";
import { formatTimestamp } from "../utils";
import { baseProps } from "./base";
import type { PullRequest, PullRequestComment } from "./types";
import { addDetailIfPresent, buildBitbucketExecutionSubtitle, defaultOutput, shortHash } from "./utils";

/**
 * The three pull request write components emit the same payload, so they share a
 * mapper and only differ in the wording used when there is nothing to show yet.
 */
function pullRequestMapper(fallbackLabel: string): ComponentBaseMapper {
  return {
    props(context: ComponentBaseContext): ComponentBaseProps {
      return baseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);
    },

    subtitle(context: SubtitleContext): string | React.ReactNode {
      const output = defaultOutput<PullRequest>(context.execution.outputs);

      if (output) {
        return `#${output.data.id ?? ""} ${output.data.title || ""}`.trim();
      }

      return buildBitbucketExecutionSubtitle(context.execution, fallbackLabel);
    },

    getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
      const output = defaultOutput<PullRequest>(context.execution.outputs);

      if (!output) {
        return {};
      }

      const pullRequest = output.data;
      const details: Record<string, string> = {
        "Created At": formatTimestamp(pullRequest.created_on, output.timestamp),
        "Pull Request": pullRequest.id ? `#${pullRequest.id} ${pullRequest.title || ""}`.trim() : "-",
      };

      addDetailIfPresent(details, "Pull Request URL", pullRequest.links?.html?.href);
      addDetailIfPresent(details, "Source Branch", pullRequest.source?.branch?.name);
      addDetailIfPresent(details, "Target Branch", pullRequest.destination?.branch?.name);
      addDetailIfPresent(details, "State", pullRequest.state);
      addDetailIfPresent(details, "Merge Commit", shortHash(pullRequest.merge_commit?.hash));
      addDetailIfPresent(details, "Closed By", pullRequest.closed_by?.display_name);

      return details;
    },
  };
}

export const createPullRequestMapper = pullRequestMapper("Pull Request Created");
export const updatePullRequestMapper = pullRequestMapper("Pull Request Updated");
export const mergePullRequestMapper = pullRequestMapper("Pull Request Merged");

export const createPRCommentMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return baseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    const output = defaultOutput<PullRequestComment>(context.execution.outputs);

    if (output) {
      return output.data.content?.raw?.trim() || "Comment Created";
    }

    return buildBitbucketExecutionSubtitle(context.execution, "Comment Created");
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const output = defaultOutput<PullRequestComment>(context.execution.outputs);

    if (!output) {
      return {};
    }

    const comment = output.data;
    const details: Record<string, string> = {
      "Created At": formatTimestamp(comment.created_on, output.timestamp),
    };

    addDetailIfPresent(details, "Comment", comment.content?.raw?.trim());
    addDetailIfPresent(details, "Comment URL", comment.links?.html?.href);
    addDetailIfPresent(details, "Author", comment.user?.display_name);

    return details;
  },
};
