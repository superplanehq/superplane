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

interface DeletedReleaseOutput {
  id?: number;
  tag_name?: string;
  name?: string;
  html_url?: string;
  draft?: boolean;
  prerelease?: boolean;
  deleted_at?: string;
  tag_deleted?: boolean;
}

export const deleteReleaseMapper: ComponentBaseMapper = {
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

    if (outputs && outputs.default && outputs.default.length > 0) {
      const deletedRelease = outputs.default[0].data as DeletedReleaseOutput;
      Object.assign(details, {
        "Deleted At": deletedRelease?.deleted_at ? new Date(deletedRelease.deleted_at).toLocaleString() : "-",
        "Tag Deleted": deletedRelease?.tag_deleted ? "Yes" : "No",
      });

      details["Release ID"] = deletedRelease?.id?.toString() || "";
      details["Tag Name"] = deletedRelease?.tag_name || "";

      if (deletedRelease?.name) {
        details["Release Name"] = deletedRelease.name;
      }

      if (deletedRelease?.draft) {
        details["Was Draft"] = "Yes";
      }

      if (deletedRelease?.prerelease) {
        details["Was Prerelease"] = "Yes";
      }
    }

    return details;
  },
};
