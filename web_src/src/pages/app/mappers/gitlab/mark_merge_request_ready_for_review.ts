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
import type { GitLabNodeMetadata, MergeRequest } from "./types";
import { buildGitlabExecutionSubtitle } from "./utils";

interface MarkMergeRequestReadyForReviewConfiguration {
  project?: string;
  mergeRequestIid?: string;
}

export const markMergeRequestReadyForReviewMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const props = baseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);
    const configuration = (context.node.configuration as MarkMergeRequestReadyForReviewConfiguration | undefined) ?? {};
    const metadata = (context.node.metadata as GitLabNodeMetadata | undefined) ?? ({} as GitLabNodeMetadata);

    const project = metadata?.project?.name || configuration.project;
    const metadataItems: MetadataItem[] = [];

    if (project) {
      metadataItems.push({ icon: "book", label: project });
    }

    if (configuration.mergeRequestIid) {
      metadataItems.push({ icon: "git-merge", label: `!${configuration.mergeRequestIid}` });
    }

    return {
      ...props,
      metadata: metadataItems.length > 0 ? metadataItems : props.metadata,
    };
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    return buildGitlabExecutionSubtitle(context.execution);
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const payload = outputs?.default?.[0];

    if (!payload) {
      return {};
    }

    const mergeRequest = (payload.data ?? {}) as MergeRequest;

    return {
      "Executed At": context.execution.createdAt ? new Date(context.execution.createdAt).toLocaleString() : "-",
      "Merge Request": mergeRequest.iid ? `!${mergeRequest.iid} ${mergeRequest.title || ""}`.trim() : "-",
      "Merge Request URL": stringOrDash(mergeRequest.web_url),
      "Ready for Review": formatReadyForReview(mergeRequest.draft),
    };
  },
};

function formatReadyForReview(draft: boolean | undefined): string {
  if (draft === undefined) {
    return "-";
  }

  return draft ? "No" : "Yes";
}
