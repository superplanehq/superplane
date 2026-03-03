import { ComponentBaseProps } from "@/ui/componentBase";
import {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  OutputPayload,
  SubtitleContext,
} from "../types";
import { baseProps } from "./base";
import { buildGithubExecutionSubtitle } from "./utils";
import { Comment } from "./types";
import { BaseNodeMetadata } from "./types";

interface CreateIssueCommentConfiguration {
  repository?: string;
  issueNumber?: string;
}

export const createIssueCommentMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const props = baseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);
    const configuration = (context.node.configuration as CreateIssueCommentConfiguration | undefined) ?? {};
    const metadata = (context.node.metadata as BaseNodeMetadata | undefined) ?? ({} as BaseNodeMetadata);
    const metadataItems = [];

    const repository = configuration.repository || metadata?.repository?.name;
    if (repository) {
      metadataItems.push({
        icon: "book",
        label: repository,
      });
    }

    if (configuration.issueNumber) {
      metadataItems.push({
        icon: "hash",
        label: `Issue: ${configuration.issueNumber}`,
      });
    }

    return {
      ...props,
      metadata: metadataItems,
    };
  },
  subtitle(context: SubtitleContext): string {
    return buildGithubExecutionSubtitle(context.execution);
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const details: Record<string, string> = {};

    if (!outputs?.default || outputs.default.length === 0) {
      return details;
    }

    const comment = outputs.default[0].data as Comment;
    details["Created At"] = comment?.created_at ? new Date(comment.created_at).toLocaleString() : "-";
    details["URL"] = comment?.html_url || "-";

    return details;
  },
};
