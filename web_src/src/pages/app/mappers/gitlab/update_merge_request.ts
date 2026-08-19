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

interface UpdateMergeRequestConfiguration {
  project?: string;
  mergeRequestIid?: string;
  state?: string;
}

export const updateMergeRequestMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const props = baseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);
    const configuration = (context.node.configuration as UpdateMergeRequestConfiguration | undefined) ?? {};
    const metadata = (context.node.metadata as GitLabNodeMetadata | undefined) ?? ({} as GitLabNodeMetadata);

    const project = metadata?.project?.name || configuration.project;
    const metadataItems: MetadataItem[] = [];

    if (project) {
      metadataItems.push({ icon: "book", label: project });
    }

    if (configuration.mergeRequestIid) {
      metadataItems.push({ icon: "git-merge", label: `!${configuration.mergeRequestIid}` });
    }

    if (configuration.state) {
      metadataItems.push({ icon: "refresh-cw", label: configuration.state === "close" ? "Close" : "Reopen" });
    }

    return {
      ...props,
      metadata: metadataItems.length > 0 ? metadataItems : props.metadata,
    };
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    if (outputs?.default?.[0]?.data) {
      const mergeRequest = outputs.default[0].data as MergeRequest;
      return `!${mergeRequest.iid} ${mergeRequest.title}`.trim();
    }
    return buildGitlabExecutionSubtitle(context.execution, "Merge Request Updated");
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const payload = outputs?.default?.[0];

    if (!payload?.data) {
      return {};
    }

    const mergeRequest = payload.data as MergeRequest;
    const details: Record<string, string> = {
      "Updated At": formatTimestamp(mergeRequest.updated_at, payload.timestamp),
      "Merge Request": mergeRequest.iid ? `!${mergeRequest.iid} ${mergeRequest.title || ""}`.trim() : "-",
    };

    addDetailIfPresent(details, "Merge Request URL", mergeRequest.web_url);
    addDetailIfPresent(details, "Source Branch", mergeRequest.source_branch);
    addDetailIfPresent(details, "Target Branch", mergeRequest.target_branch);
    addDetailIfPresent(details, "State", mergeRequest.state);

    return details;
  },
};

function addDetailIfPresent(details: Record<string, string>, label: string, value?: string) {
  if (value) {
    details[label] = value;
  }
}
