import { ComponentBaseProps } from "@/ui/componentBase";
import {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  OutputPayload,
  SubtitleContext,
} from "../types";
import { buildExecutionSubtitle, stringOrDash } from "../utils";
import { baseProps } from "./base";
import { Comment } from "./types";

export const createIssueCommentMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return baseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);
  },
  subtitle(context: SubtitleContext): string {
    return buildExecutionSubtitle(context.execution);
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const details: Record<string, string> = {};

    if (!outputs?.default || outputs.default.length === 0) {
      return details;
    }

    const comment = outputs.default[0].data as Comment;
    details["ID"] = stringOrDash(comment?.id);
    details["Body"] = stringOrDash(comment?.content?.raw);
    details["Author"] = stringOrDash(comment?.user?.display_name);
    details["Created At"] = comment?.created_on ? new Date(comment.created_on).toLocaleString() : "-";
    details["URL"] = stringOrDash(comment?.links?.html?.href);

    return details;
  },
};
