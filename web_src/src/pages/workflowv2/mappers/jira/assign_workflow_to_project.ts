import type { ComponentBaseProps } from "@/ui/componentBase";
import type React from "react";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import type { MetadataItem } from "@/ui/metadataList";
import { renderTimeAgo } from "@/components/TimeAgo";
import { jiraComponentBaseProps } from "./base";
import { addDetail, addProjectMetadata } from "./utils";
import type { AssignWorkflowToProjectConfiguration, JiraNodeMetadata, JiraWorkflowSchemeAssignment } from "./types";

export const assignWorkflowToProjectMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return jiraComponentBaseProps(context, metadataList(context.node));
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const details: Record<string, string> = {
      "Executed At": context.execution.createdAt ? new Date(context.execution.createdAt).toLocaleString() : "-",
    };

    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const assignment = outputs?.default?.[0]?.data as JiraWorkflowSchemeAssignment | undefined;
    if (assignment) {
      addDetail(details, "Project ID", assignment.projectId);
      addDetail(details, "Workflow Scheme ID", assignment.workflowSchemeId);
      details["Draft Created"] = assignment.draftCreated ? "Yes" : "No";
      if (assignment.dryRun) details["Dry Run"] = "Yes";
      addDetail(details, "Task ID", assignment.taskId);
      addDetail(details, "Task Status", assignment.taskStatus);
      addDetail(details, "Task URL", assignment.taskSelf);
    }

    return details;
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const assignment = outputs?.default?.[0]?.data as JiraWorkflowSchemeAssignment | undefined;
    if (assignment?.taskStatus) return assignment.taskStatus;
    if (assignment?.dryRun) return "Dry run";
    if (context.execution.createdAt) {
      return renderTimeAgo(new Date(context.execution.createdAt));
    }
    return "";
  },
};

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const nodeMetadata = node.metadata as JiraNodeMetadata | undefined;
  const configuration = node.configuration as AssignWorkflowToProjectConfiguration | undefined;

  addProjectMetadata(metadata, nodeMetadata?.project, configuration?.project);

  const schemeLabel = nodeMetadata?.workflowScheme?.name || configuration?.workflowScheme;
  if (schemeLabel) {
    metadata.push({ icon: "workflow", label: schemeLabel });
  }

  if (configuration?.dryRun) {
    metadata.push({ icon: "search-check", label: "Dry run" });
  }

  return metadata;
}
