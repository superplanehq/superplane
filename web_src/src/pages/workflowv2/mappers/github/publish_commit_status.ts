import {
  ComponentsNode,
  ComponentsComponent,
  WorkflowsWorkflowNodeExecution,
  WorkflowsWorkflowNodeQueueItem,
} from "@/api-client";
import { ComponentBaseProps } from "@/ui/componentBase";
import { ComponentBaseMapper, OutputPayload } from "../types";
import { baseProps } from "./base";

interface CommitStatus {
  id?: number;
  state?: string;
  context?: string;
  description?: string;
  target_url?: string;
  creator?: {
    login?: string;
  };
  created_at?: string;
  updated_at?: string;
}

export const publishCommitStatusMapper: ComponentBaseMapper = {
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

    const status = outputs.default[0].data as CommitStatus;

    const details: Record<string, string> = {
      State: status?.state || "-",
      Context: status?.context || "-",
      Description: status?.description || "-",
      "Target URL": status?.target_url || "-",
      "Status ID": status?.id?.toString() || "-",
      "Created At": status?.created_at || "-",
    };

    if (status?.creator?.login) {
      details["Created By"] = status.creator.login;
    }

    return details;
  },
};
