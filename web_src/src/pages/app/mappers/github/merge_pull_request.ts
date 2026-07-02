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

interface MergePullRequestConfiguration {
  repository?: string;
  pullNumber?: string | number;
  mergeMethod?: string;
}

interface PullRequestMergeResult {
  sha?: string;
  merged?: boolean;
  message?: string;
}

export const mergePullRequestMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return baseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    return buildGithubExecutionSubtitle(context.execution);
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const configuration = mergePullRequestConfiguration(context);
    const result = mergePullRequestOutput(context);
    const repositoryURL = repositoryUrl(context);
    const pullNumber = configuration.pullNumber;

    return {
      "Created At": context.execution.createdAt ? new Date(context.execution.createdAt).toLocaleString() : "-",
      Repository: stringOrDash(configuration.repository || repositoryName(context)),
      "Pull Request": formatPullRequestNumber(pullNumber),
      "Pull Request URL": stringOrDash(pullRequestUrl(repositoryURL, pullNumber)),
      "Merge method": formatMergeMethod(configuration.mergeMethod),
      Merged: formatMerged(result?.merged),
      SHA: stringOrDash(shortSha(result?.sha)),
      Message: stringOrDash(result?.message),
    };
  },
};

function mergePullRequestConfiguration(context: ExecutionDetailsContext): MergePullRequestConfiguration {
  return (
    ((context.execution.configuration || context.node.configuration || {}) as
      | MergePullRequestConfiguration
      | undefined) || {}
  );
}

function mergePullRequestOutput(context: ExecutionDetailsContext): PullRequestMergeResult | undefined {
  const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
  return outputs?.default?.[0]?.data as PullRequestMergeResult | undefined;
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

function formatMergeMethod(value: string | undefined): string {
  switch (value) {
    case "squash":
      return "Squash";
    case "rebase":
      return "Rebase";
    case "merge":
    case "":
    case undefined:
      return "Merge commit";
    default:
      return value;
  }
}

function formatMerged(value: boolean | undefined): string {
  if (value === undefined) {
    return "-";
  }

  return value ? "Yes" : "No";
}

function shortSha(sha: string | undefined): string | undefined {
  return sha ? sha.slice(0, 7) : undefined;
}
