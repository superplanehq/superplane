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
import { addDetail, addIssueKeyMetadata, addProjectMetadata } from "./utils";
import type { GetWorkflowConfiguration, JiraNodeMetadata, JiraWorkflow } from "./types";

export const getWorkflowMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return jiraComponentBaseProps(context, metadataList(context.node));
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const details: Record<string, string> = {
      "Executed At": context.execution.createdAt ? new Date(context.execution.createdAt).toLocaleString() : "-",
    };

    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const workflow = outputs?.default?.[0]?.data as JiraWorkflow | undefined;
    if (workflow) {
      addDetail(details, "Issue", workflow.issueKey);
      addDetail(details, "Issue Type", workflow.issueType);
      addDetail(details, "Current Status", workflow.currentStatus);
      addDetail(details, "Workflow", workflow.workflowName);
      addDetail(details, "Workflow Scheme", workflow.workflowSchemeName);
      if (workflow.availableTransitions?.length) {
        details["Available Transitions"] = workflow.availableTransitions
          .map((t) => t.toStatus || t.name)
          .filter(Boolean)
          .join(", ");
      }
    }

    return details;
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    const timestamp = context.execution.updatedAt || context.execution.createdAt;
    return timestamp ? renderTimeAgo(new Date(timestamp)) : "";
  },
};

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const nodeMetadata = node.metadata as JiraNodeMetadata | undefined;
  const configuration = node.configuration as GetWorkflowConfiguration | undefined;

  addProjectMetadata(metadata, nodeMetadata?.project, configuration?.project);
  addIssueKeyMetadata(metadata, "hash", configuration?.issueKey);

  return metadata;
}
