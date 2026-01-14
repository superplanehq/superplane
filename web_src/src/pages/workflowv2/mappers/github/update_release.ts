import {
  ComponentsNode,
  ComponentsComponent,
  WorkflowsWorkflowNodeExecution,
  WorkflowsWorkflowNodeQueueItem,
} from "@/api-client";
import { ComponentBaseProps } from "@/ui/componentBase";
import { ComponentBaseMapper, OutputPayload } from "../types";
import { baseProps } from "./base";

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

export const updateReleaseMapper: ComponentBaseMapper = {
  props(
    nodes: ComponentsNode[],
    node: ComponentsNode,
    componentDefinition: ComponentsComponent,
    lastExecutions: WorkflowsWorkflowNodeExecution[],
    queueItems: WorkflowsWorkflowNodeQueueItem[],
  ): ComponentBaseProps {
    return baseProps(nodes, node, componentDefinition, lastExecutions, queueItems);
  },

  getExecutionDetails(execution: WorkflowsWorkflowNodeExecution, _node: ComponentsNode): Record<string, string> {
    const outputs = execution.outputs as { default?: OutputPayload[] } | undefined;

    // If no outputs (e.g., execution failed), return empty details
    if (!outputs || !outputs.default || outputs.default.length === 0) {
      return {};
    }

    const release = outputs.default[0].data as ReleaseOutput;

    const details: Record<string, string> = {
      URL: release?.html_url || "-",
      "Release ID": release?.id?.toString() || "-",
      "Tag Name": release?.tag_name || "-",
    };

    if (release?.name) {
      details["Name"] = release.name;
    }

    if (release?.draft !== undefined) {
      details["Draft"] = release.draft ? "Yes" : "No";
    }

    if (release?.prerelease !== undefined) {
      details["Prerelease"] = release.prerelease ? "Yes" : "No";
    }

    if (release?.published_at) {
      details["Published At"] = release.published_at;
    }

    if (release?.author?.login) {
      details["Updated By"] = release.author.login;
    }

    return details;
  },
};
