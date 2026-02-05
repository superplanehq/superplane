import { ComponentBaseProps } from "@/ui/componentBase";
import {
  ComponentBaseMapper,
  ComponentBaseContext,
  SubtitleContext,
  ExecutionDetailsContext,
  OutputPayload,
  NodeInfo,
} from "../types";
import { baseProps } from "./base";
import { buildGithubExecutionSubtitle } from "./utils";
import { MetadataItem } from "@/ui/metadataList";

interface CommentOutput {
  id?: number;
  node_id?: string;
  body?: string;
  html_url?: string;
  issue_url?: string;
  created_at?: string;
  updated_at?: string;
  user?: {
    login?: string;
    html_url?: string;
  };
}

interface CreateIssueCommentConfiguration {
  repository?: string;
  issueNumber?: string;
  body?: string;
}

function getCreateIssueCommentMetadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as CreateIssueCommentConfiguration | undefined;
  const nodeMetadata = node.metadata as { repository?: { name?: string } } | undefined;

  if (nodeMetadata?.repository?.name) {
    metadata.push({ icon: "book", label: nodeMetadata.repository.name });
  }

  if (configuration?.issueNumber) {
    metadata.push({ icon: "hash", label: `Issue #${configuration.issueNumber}` });
  }

  return metadata;
}

export const createIssueCommentMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const base = baseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);

    return {
      ...base,
      metadata: getCreateIssueCommentMetadataList(context.node),
    };
  },

  subtitle(context: SubtitleContext): string {
    return buildGithubExecutionSubtitle(context.execution);
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const details: Record<string, string> = {};

    Object.assign(details, {
      "Created At": context.execution.createdAt ? new Date(context.execution.createdAt).toLocaleString() : "-",
    });

    if (outputs?.default && outputs.default.length > 0) {
      const comments = outputs.default[0].data as CommentOutput | CommentOutput[];
      const comment = Array.isArray(comments) ? comments[0] : comments;

      if (comment) {
        if (comment.html_url) {
          details["Comment URL"] = comment.html_url;
        }

        if (comment.user?.login) {
          details["Author"] = comment.user.html_url || comment.user.login;
        }

        if (comment.body) {
          const truncated = comment.body.length > 100 ? comment.body.substring(0, 100) + "..." : comment.body;
          details["Body"] = truncated;
        }

        if (comment.id) {
          details["Comment ID"] = comment.id.toString();
        }
      }
    }

    return details;
  },
};
