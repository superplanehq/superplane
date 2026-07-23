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
import type { GitLabNodeMetadata, Note } from "./types";
import { buildGitlabExecutionSubtitle } from "./utils";

interface UpdateIssueCommentConfiguration {
  project?: string;
  issueIid?: string;
  commentId?: string;
}

export const updateIssueCommentMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const props = baseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);
    const configuration = (context.node.configuration as UpdateIssueCommentConfiguration | undefined) ?? {};
    const metadata = (context.node.metadata as GitLabNodeMetadata | undefined) ?? ({} as GitLabNodeMetadata);

    const project = metadata?.project?.name || configuration.project;
    const metadataItems: MetadataItem[] = [];

    if (project) {
      metadataItems.push({ icon: "book", label: project });
    }

    if (configuration.issueIid) {
      metadataItems.push({ icon: "circle-dot", label: `#${configuration.issueIid}` });
    }

    return {
      ...props,
      metadata: metadataItems.length > 0 ? metadataItems : props.metadata,
    };
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    return buildGitlabExecutionSubtitle(context.execution, "Comment Updated");
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const payload = outputs?.default?.[0];

    if (!payload?.data) {
      return {};
    }

    const note = payload.data as Note;
    const details: Record<string, string> = {
      "Updated At": formatTimestamp(note.updated_at, payload.timestamp),
    };

    if (note.id) {
      details["Comment ID"] = String(note.id);
    }

    details["Updated By"] = note.author?.username || "-";

    return details;
  },
};
