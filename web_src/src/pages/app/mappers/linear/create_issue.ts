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
import { linearComponentBaseProps } from "./base";
import { addDetail, addTeamMetadata, getIssueLabel, getUserLabel } from "./utils";
import type { CreateIssueConfiguration, LinearIssue, LinearNodeMetadata } from "./types";

export const createIssueMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return linearComponentBaseProps(context, metadataList(context.node));
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const details: Record<string, string> = {
      "Executed At": context.execution.createdAt ? new Date(context.execution.createdAt).toLocaleString() : "-",
    };

    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const issue = outputs?.default?.[0]?.data as LinearIssue | undefined;
    if (!issue) return details;

    addDetail(details, "Issue", issue.identifier);
    addDetail(details, "Issue URL", issue.url);
    addDetail(details, "Title", issue.title);
    addDetail(details, "Status", issue.state?.name);
    addDetail(details, "Assignee", getUserLabel(issue.assignee));

    return details;
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const issue = outputs?.default?.[0]?.data as LinearIssue | undefined;

    const label = getIssueLabel(issue);
    if (label) return label;

    if (context.execution.createdAt) {
      return renderTimeAgo(new Date(context.execution.createdAt));
    }

    return "";
  },
};

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const nodeMetadata = node.metadata as LinearNodeMetadata | undefined;
  const configuration = node.configuration as CreateIssueConfiguration | undefined;

  addTeamMetadata(metadata, nodeMetadata?.team, configuration?.team);

  const priority = priorityLabel(configuration?.priority);
  if (priority) {
    metadata.push({ icon: "flag", label: priority });
  }

  return metadata;
}

/** Linear encodes priority as 0-4; the picker stores the number as a string. */
const PRIORITY_LABELS: Record<string, string> = {
  "0": "No priority",
  "1": "Urgent",
  "2": "High",
  "3": "Medium",
  "4": "Low",
};

function priorityLabel(priority: string | undefined): string | undefined {
  if (!priority) return undefined;
  return PRIORITY_LABELS[priority];
}
