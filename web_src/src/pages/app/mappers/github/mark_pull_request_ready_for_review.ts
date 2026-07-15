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
import type { BaseNodeMetadata } from "./types";
import { buildGithubExecutionSubtitle } from "./utils";

interface MarkPullRequestReadyForReviewConfiguration {
  repository?: string;
  pullNumber?: string | number;
}

interface PullRequestOutput {
  number?: number;
  title?: string;
  draft?: boolean;
  html_url?: string;
}

export const markPullRequestReadyForReviewMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return baseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    return buildGithubExecutionSubtitle(context.execution);
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const configuration = markPullRequestReadyForReviewConfiguration(context);
    const pullRequest = markPullRequestReadyForReviewOutput(context);
    const repositoryURL = repositoryUrl(context);
    const pullNumber = configuration.pullNumber ?? pullRequest?.number;

    return {
      "Created At": context.execution.createdAt ? new Date(context.execution.createdAt).toLocaleString() : "-",
      Repository: stringOrDash(configuration.repository || repositoryName(context)),
      "Pull Request": formatPullRequestNumber(pullNumber),
      "Pull Request URL": stringOrDash(pullRequest?.html_url || pullRequestUrl(repositoryURL, pullNumber)),
      Title: stringOrDash(pullRequest?.title),
      "Ready for Review": formatReadyForReview(pullRequest?.draft),
    };
  },
};

function markPullRequestReadyForReviewConfiguration(
  context: ExecutionDetailsContext,
): MarkPullRequestReadyForReviewConfiguration {
  return (
    ((context.execution.configuration || context.node.configuration || {}) as
      | MarkPullRequestReadyForReviewConfiguration
      | undefined) || {}
  );
}

function markPullRequestReadyForReviewOutput(context: ExecutionDetailsContext): PullRequestOutput | undefined {
  const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
  return outputs?.default?.[0]?.data as PullRequestOutput | undefined;
}

function repositoryName(context: ExecutionDetailsContext): string | undefined {
  const metadata = context.node.metadata as BaseNodeMetadata | undefined;
  return metadata?.repository?.name;
}

function repositoryUrl(context: ExecutionDetailsContext): string | undefined {
  const metadata = context.node.metadata as BaseNodeMetadata | undefined;
  return metadata?.repository?.url;
}

function formatPullRequestNumber(value: string | number | undefined): string {
  if (value === undefined || value === "") {
    return "-";
  }

  const text = String(value);
  return text.startsWith("#") ? text : `#${text}`;
}

function pullRequestUrl(
  repositoryURL: string | undefined,
  pullNumber: string | number | undefined,
): string | undefined {
  if (!repositoryURL || pullNumber === undefined || pullNumber === "") {
    return undefined;
  }

  return `${repositoryURL}/pull/${String(pullNumber).replace(/^#/, "")}`;
}

function formatReadyForReview(draft: boolean | undefined): string {
  if (draft === undefined) {
    return "-";
  }

  return draft ? "No" : "Yes";
}
