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
import type { GitLabNodeMetadata, MergeRequest } from "./types";
import { buildGitlabExecutionSubtitle } from "./utils";

interface AcceptMergeRequestConfiguration {
  project?: string;
  mergeRequestIid?: string;
}

export const acceptMergeRequestMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const props = baseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);
    const configuration = (context.node.configuration as AcceptMergeRequestConfiguration | undefined) ?? {};
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

    return buildMergeRequestDetails((payload.data ?? {}) as MergeRequest, payload.timestamp);
  },
};

function buildMergeRequestDetails(mergeRequest: MergeRequest, payloadTimestamp?: string): Record<string, string> {
  const details: Record<string, string> = {
    "Merged At": formatTimestamp(mergeRequest.merged_at, payloadTimestamp),
    "Merge Request": mergeRequest.iid ? `!${mergeRequest.iid} ${mergeRequest.title || ""}`.trim() : "-",
  };

  const sha = mergeRequest.merge_commit_sha || mergeRequest.squash_commit_sha;
  addDetailIfPresent(details, "Merge Request URL", mergeRequest.web_url);
  addDetailIfPresent(details, "Source Branch", mergeRequest.source_branch);
  addDetailIfPresent(details, "Target Branch", mergeRequest.target_branch);
  addDetailIfPresent(details, "Merge Commit SHA", sha?.slice(0, 8));

  return details;
}

function addDetailIfPresent(details: Record<string, string>, label: string, value?: string) {
  if (value) {
    details[label] = value;
  }
}
