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

interface ReviewOutput {
  id?: number;
  node_id?: string;
  html_url?: string;
  body?: string;
  state?: string;
  commit_id?: string;
  submitted_at?: string;
  user?: string;
  author_association?: string;
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

    if (outputs && outputs.default && outputs.default.length > 0) {
      const review = outputs.default[0].data as ReviewOutput;
      Object.assign(details, {
        "Submitted At": review?.submitted_at ? new Date(review.submitted_at).toLocaleString() : "-",
        "Submitted By": review?.user || "-",
      });

      details["Review URL"] = review?.html_url || "";
      details["Review ID"] = review?.id?.toString() || "";
      details["State"] = review?.state || "";

      if (review?.body) {
        details["Body"] = review.body;
      }

      if (review?.commit_id) {
        details["Commit ID"] = review.commit_id;
      }

      if (review?.author_association) {
        details["Author Association"] = review.author_association;
      }
    }

    return details;
  },
};
