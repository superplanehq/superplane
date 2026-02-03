import {
  ComponentsNode,
  ComponentsComponent,
  CanvasesCanvasNodeExecution,
  CanvasesCanvasNodeQueueItem,
} from "@/api-client";
import { ComponentBaseProps } from "@/ui/componentBase";
import { ComponentBaseMapper, OutputPayload } from "../types";
import { baseProps } from "./base";
import { buildGithubExecutionSubtitle } from "./utils";

interface ReviewOutput {
  id?: number;
  node_id?: string;
  state?: string;
  body?: string;
  html_url?: string;
  pull_request?: string;
  submitted_at?: string;
  user?: {
    login?: string;
    html_url?: string;
  };
}

export const createReviewMapper: ComponentBaseMapper = {
  props(
    nodes: ComponentsNode[],
    node: ComponentsNode,
    componentDefinition: ComponentsComponent,
    lastExecutions: CanvasesCanvasNodeExecution[],
    queueItems: CanvasesCanvasNodeQueueItem[],
  ): ComponentBaseProps {
    return baseProps(nodes, node, componentDefinition, lastExecutions, queueItems);
  },
  subtitle(_node: ComponentsNode, execution: CanvasesCanvasNodeExecution): string {
    const outputs = execution.outputs as { default?: OutputPayload[] } | undefined;
    if (outputs?.default && outputs.default.length > 0) {
      const reviews = outputs.default[0].data as ReviewOutput | ReviewOutput[];
      const review = Array.isArray(reviews) ? reviews[0] : reviews;
      if (review?.state) {
        return buildGithubExecutionSubtitle(execution, review.state);
      }
    }
    return buildGithubExecutionSubtitle(execution);
  },

  getExecutionDetails(execution: CanvasesCanvasNodeExecution, _node: ComponentsNode): Record<string, string> {
    const outputs = execution.outputs as { default?: OutputPayload[] } | undefined;
    const details: Record<string, string> = {};

    Object.assign(details, {
      "Submitted At": execution.createdAt ? new Date(execution.createdAt).toLocaleString() : "-",
    });

    if (outputs?.default && outputs.default.length > 0) {
      const reviews = outputs.default[0].data as ReviewOutput | ReviewOutput[];
      const review = Array.isArray(reviews) ? reviews[0] : reviews;

      if (review) {
        if (review.state) {
          details["State"] = review.state;
        }

        if (review.html_url) {
          details["Review URL"] = review.html_url;
        }

        if (review.user?.login) {
          details["Reviewer"] = review.user.html_url || review.user.login;
        }

        if (review.body) {
          // Truncate body for display
          const truncated = review.body.length > 100 ? review.body.substring(0, 100) + "..." : review.body;
          details["Body"] = truncated;
        }

        if (review.id) {
          details["Review ID"] = review.id.toString();
        }
      }
    }

    return details;
  },
};
