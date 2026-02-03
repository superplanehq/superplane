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

export const createIssueCommentMapper: ComponentBaseMapper = {
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
    return buildGithubExecutionSubtitle(execution);
  },

  getExecutionDetails(execution: CanvasesCanvasNodeExecution, _node: ComponentsNode): Record<string, string> {
    const outputs = execution.outputs as { default?: OutputPayload[] } | undefined;
    const details: Record<string, string> = {};

    Object.assign(details, {
      "Created At": execution.createdAt ? new Date(execution.createdAt).toLocaleString() : "-",
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
          // Truncate body for display
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
