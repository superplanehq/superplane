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
import type { Commit, GitLabNodeMetadata } from "./types";
import { buildGitlabExecutionSubtitle, shortSha } from "./utils";

interface GetCommitStatusConfiguration {
  project?: string;
  ref?: string;
}

export const getCommitStatusMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const props = baseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);
    const configuration = (context.node.configuration as GetCommitStatusConfiguration | undefined) ?? {};
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
    const commit = outputs?.default?.[0]?.data as Commit | undefined;
    return buildGitlabExecutionSubtitle(context.execution, commit?.status);
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const payload = outputs?.default?.[0];

    if (!payload?.data) {
      return {};
    }

    const commit = payload.data as Commit;
    const pipeline = commit.last_pipeline;

    const details: Record<string, string> = {
      "Retrieved At": formatTimestamp(undefined, payload.timestamp),
      Status: commit.status || "-",
      Commit: commit.short_id || (commit.id ? shortSha(commit.id) : "-"),
    };

    if (pipeline?.status) {
      details["Pipeline"] = pipeline.status;
    }

    if (pipeline?.ref) {
      details["Ref"] = pipeline.ref;
    }

    const url = pipeline?.web_url || commit.web_url;
    if (url) {
      details["URL"] = url;
    }

    return details;
  },
};
