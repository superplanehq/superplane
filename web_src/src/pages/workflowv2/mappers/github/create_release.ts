import {
  ComponentsNode,
  ComponentsComponent,
  WorkflowsWorkflowNodeExecution,
  WorkflowsWorkflowNodeQueueItem,
} from "@/api-client";
import { ComponentBaseProps } from "@/ui/componentBase";
import { ComponentBaseMapper, OutputPayload } from "../types";
import { baseProps } from "./base";
import { buildGithubExecutionSubtitle } from "./utils";

interface ReleaseOutput {
  id?: number;
  tag_name?: string;
  name?: string;
  html_url?: string;
  draft?: boolean;
  prerelease?: boolean;
  created_at?: string;
  published_at?: string;
  author?: {
    login?: string;
  };
}

export const createReleaseMapper: ComponentBaseMapper = {
  props(
    nodes: ComponentsNode[],
    node: ComponentsNode,
    componentDefinition: ComponentsComponent,
    lastExecutions: WorkflowsWorkflowNodeExecution[],
    queueItems: WorkflowsWorkflowNodeQueueItem[],
  ): ComponentBaseProps {
    return baseProps(nodes, node, componentDefinition, lastExecutions, queueItems);
  },
  subtitle(_node: ComponentsNode, execution: WorkflowsWorkflowNodeExecution): string {
    return buildGithubExecutionSubtitle(execution);
  },

  getExecutionDetails(execution: WorkflowsWorkflowNodeExecution, _node: ComponentsNode): Record<string, string> {
    const outputs = execution.outputs as { default?: OutputPayload[] } | undefined;
    const details: Record<string, string> = {};

    if (execution.createdAt) {
      details["Started At"] = execution.createdAt;
    }

    if (execution.state === "STATE_FINISHED" && execution.updatedAt) {
      details["Finished At"] = execution.updatedAt;
    }

    if (!outputs || !outputs.default || outputs.default.length === 0) {
      return details;
    }

    const release = outputs.default[0].data as ReleaseOutput;

    details["Release URL"] = release?.html_url || "";
    details["Release ID"] = release?.id?.toString() || "";
    details["Tag Name"] = release?.tag_name || "";

    if (release?.name) {
      details["Release Name"] = release.name;
    }

    if (release?.draft !== undefined) {
      details["Draft"] = release.draft ? "Yes" : "No";
    }

    if (release?.prerelease !== undefined) {
      details["Prerelease"] = release.prerelease ? "Yes" : "No";
    }

    if (release?.created_at) {
      details["Created At"] = release.created_at;
    }

    if (release?.author?.login) {
      details["Created By"] = release.author.login;
    }

    return details;
  },
};
