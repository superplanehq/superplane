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
import { addDetail, addIssueKeyMetadata } from "./utils";
import type { ApproveWorkflowConfiguration, JiraApproval } from "./types";

export const approveWorkflowMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return jiraComponentBaseProps(context, metadataList(context.node));
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const details: Record<string, string> = {
      "Executed At": context.execution.createdAt ? new Date(context.execution.createdAt).toLocaleString() : "-",
    };

    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const approval = outputs?.default?.[0]?.data as JiraApproval | undefined;
    if (approval) {
      addDetail(details, "Approval ID", approval.id);
      addDetail(details, "Name", approval.name);
      addDetail(details, "Decision", approval.finalDecision);
      if (approval.approvers?.length) {
        details["Approvers"] = approval.approvers
          .map((entry) => entry.approver?.displayName)
          .filter(Boolean)
          .join(", ");
      }
    }

    const configuration = context.node.configuration as ApproveWorkflowConfiguration | undefined;
    addDetail(details, "Issue Key", configuration?.issueKey);
    addDetail(details, "Configured Decision", configuration?.decision);

    return details;
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const approval = outputs?.default?.[0]?.data as JiraApproval | undefined;
    if (approval?.finalDecision) return approval.finalDecision;
    if (context.execution.createdAt) {
      return renderTimeAgo(new Date(context.execution.createdAt));
    }
    return "";
  },
};

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as ApproveWorkflowConfiguration | undefined;

  addIssueKeyMetadata(metadata, "hash", configuration?.issueKey);

  if (configuration?.decision) {
    metadata.push({
      icon: configuration.decision === "approve" ? "circle-check" : "circle-x",
      label: configuration.decision,
    });
  }

  if (configuration?.approvalSelector === "byId" && configuration.approvalId) {
    metadata.push({ icon: "badge-check", label: configuration.approvalId });
  }

  return metadata;
}
