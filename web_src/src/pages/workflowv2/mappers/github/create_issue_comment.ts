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

interface IssueComment {
  id?: number;
  body?: string;
  html_url?: string;
  created_at?: string;
  updated_at?: string;
  user?: {
    login?: string;
    html_url?: string;
  };
}

export const createIssueCommentMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return baseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);
  },
  subtitle(context: SubtitleContext): string {
    return buildGithubExecutionSubtitle(context.execution);
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    let details: Record<string, string> = {};

    if (outputs && outputs.default && outputs.default.length > 0) {
      const comment = outputs.default[0].data as IssueComment;
      Object.assign(details, {
        "Created At": comment.created_at ? new Date(comment.created_at).toLocaleString() : "-",
        "Created By": comment.user?.login || "-",
      });

      if (comment.updated_at) {
        details["Updated At"] = new Date(comment.updated_at).toLocaleString();
      }

      details["Comment ID"] = comment?.id?.toString() || "";
      details["Body"] = comment?.body || "";
      details["URL"] = comment?.html_url || "";
      details["Author URL"] = comment?.user?.html_url || "";
    }

    return details;
  },
};
