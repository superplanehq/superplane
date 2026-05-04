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
import { baseProps } from "./base";
import { buildGithubExecutionSubtitle } from "./utils";
import type { BaseNodeMetadata, Comment } from "./types";

interface CreateIssueCommentConfiguration {
  repository?: string;
  issueNumber?: string;
}

function commentBodyPreview(body: string | undefined, maxLen = 120): string {
  const singleLine = (body ?? "").trim().replace(/\s+/g, " ");
  if (!singleLine) {
    return "";
  }
  if (singleLine.length <= maxLen) {
    return singleLine;
  }
  return `${singleLine.slice(0, maxLen - 1)}…`;
}

export const createIssueCommentMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const props = baseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);
    const configuration = (context.node.configuration as CreateIssueCommentConfiguration | undefined) ?? {};
    const metadata = (context.node.metadata as BaseNodeMetadata | undefined) ?? ({} as BaseNodeMetadata);

    const repository = configuration.repository || metadata?.repository?.name;
    const issueNumber = configuration.issueNumber?.trim();
    const metadataItems: MetadataItem[] = [];

    if (repository) {
      metadataItems.push({ icon: "book", label: repository });
    }
    if (issueNumber) {
      metadataItems.push({ icon: "hash", label: `Issue #${issueNumber}` });
    }

    return {
      ...props,
      metadata: metadataItems.length > 0 ? metadataItems : props.metadata,
    };
  },
  subtitle(context: SubtitleContext): string | React.ReactNode {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const comment = outputs?.default?.[0]?.data as Comment | undefined;
    const preview = commentBodyPreview(comment?.body);
    return buildGithubExecutionSubtitle(context.execution, preview);
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const details: Record<string, string> = {};

    if (!outputs?.default || outputs.default.length === 0) {
      return details;
    }

    const comment = outputs.default[0].data as Comment;
    Object.assign(details, {
      "Created At": comment?.created_at ? new Date(comment.created_at).toLocaleString() : "-",
      "Created By": comment?.user?.login || "-",
    });

    details["Comment ID"] = comment?.id != null ? String(comment.id) : "-";
    details["URL"] = comment?.html_url || "-";
    details["Author"] = comment?.user?.html_url || comment?.user?.login || "-";

    if (comment?.node_id) {
      details["Node ID"] = comment.node_id;
    }

    if (comment?.updated_at) {
      details["Updated At"] = new Date(comment.updated_at).toLocaleString();
    }

    const bodyPreview = commentBodyPreview(comment?.body, 200);
    if (bodyPreview) {
      details["Body"] = bodyPreview;
    }

    return details;
  },
};
