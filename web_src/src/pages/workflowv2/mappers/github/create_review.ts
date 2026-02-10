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

interface PullRequestReviewOutput {
  id?: number;
  state?: string;
  body?: string;
  html_url?: string;
  submitted_at?: string;
  user?: {
    login?: string;
    html_url?: string;
  };
}

export const createReviewMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return baseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);
  },

  subtitle(context: SubtitleContext): string {
    return buildGithubExecutionSubtitle(context.execution);
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const details: Record<string, string> = {};

    if (!outputs || !outputs.default || outputs.default.length === 0) {
      return details;
    }

    const review = outputs.default[0].data as PullRequestReviewOutput;
    details["Submitted At"] = review?.submitted_at ? new Date(review.submitted_at).toLocaleString() : "-";
    details["Review URL"] = review?.html_url || "";

    return details;
  },
};
