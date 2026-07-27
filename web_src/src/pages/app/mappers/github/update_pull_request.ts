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
import { stringOrDash } from "../utils";
import { baseProps } from "./base";
import type { BaseNodeMetadata } from "./types";
import { buildGithubExecutionSubtitle } from "./utils";

interface UpdatePullRequestConfiguration {
  repository?: string;
  pullNumber?: string | number;
}

interface PullRequestOutput {
  number?: number;
  title?: string;
  state?: string;
  html_url?: string;
  base?: {
    ref?: string;
  };
  labels?: { name: string }[];
  assignees?: { login: string }[];
}

export const updatePullRequestMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const props = baseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);
    const configuration = (context.node.configuration as UpdatePullRequestConfiguration | undefined) ?? {};
    const metadata = (context.node.metadata as BaseNodeMetadata | undefined) ?? ({} as BaseNodeMetadata);

    const repository = metadata?.repository?.name || configuration.repository;
    const metadataItems: MetadataItem[] = [];

    if (repository) {
      metadataItems.push({ icon: "book", label: repository });
    }

    if (configuration.pullNumber !== undefined && configuration.pullNumber !== "") {
      metadataItems.push({ icon: "git-pull-request", label: formatPullRequestNumber(configuration.pullNumber) });
    }

    return {
      ...props,
      metadata: metadataItems.length > 0 ? metadataItems : props.metadata,
    };
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    return buildGithubExecutionSubtitle(context.execution);
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const configuration = updatePullRequestConfiguration(context);
    const pullRequest = updatePullRequestOutput(context);
    const repositoryURL = repositoryUrl(context);
    const pullNumber = configuration.pullNumber ?? pullRequest?.number;

    return {
      "Created At": context.execution.createdAt ? new Date(context.execution.createdAt).toLocaleString() : "-",
      Repository: stringOrDash(configuration.repository || repositoryName(context)),
      "Pull Request": formatPullRequestNumber(pullNumber),
      "Pull Request URL": stringOrDash(pullRequest?.html_url || pullRequestUrl(repositoryURL, pullNumber)),
      Title: stringOrDash(pullRequest?.title),
      State: stringOrDash(pullRequest?.state),
      "Base Branch": stringOrDash(pullRequest?.base?.ref),
      Labels: formatList(pullRequest?.labels?.map((label) => label.name)),
      Assignees: formatList(pullRequest?.assignees?.map((assignee) => assignee.login)),
    };
  },
};

function updatePullRequestConfiguration(context: ExecutionDetailsContext): UpdatePullRequestConfiguration {
  return (
    ((context.execution.configuration || context.node.configuration || {}) as
      | UpdatePullRequestConfiguration
      | undefined) || {}
  );
}

function updatePullRequestOutput(context: ExecutionDetailsContext): PullRequestOutput | undefined {
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

function formatList(values: string[] | undefined): string {
  if (!values || values.length === 0) {
    return "-";
  }

  return values.join(", ");
}
