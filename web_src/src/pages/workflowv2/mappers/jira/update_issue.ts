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
import { addDetail, addIssueKeyMetadata, addProjectMetadata, getIssueLabel, getIssueUrl } from "./utils";
import type { JiraIssue, JiraNodeMetadata, UpdateIssueConfiguration } from "./types";

const FIELD_LABELS: Record<keyof UpdateIssueConfiguration, string> = {
  project: "Project",
  issueKey: "Issue Key",
  summary: "Summary",
  description: "Description",
  issueType: "Issue Type",
  assignee: "Assignee",
  priority: "Priority",
  labels: "Labels",
  notifyUsers: "Notify",
};

export const updateIssueMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return jiraComponentBaseProps(context, metadataList(context.node));
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const details: Record<string, string> = {
      "Executed At": context.execution.createdAt ? new Date(context.execution.createdAt).toLocaleString() : "-",
    };

    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const issue = outputs?.default?.[0]?.data as JiraIssue | undefined;
    if (issue) {
      addDetail(details, "Key", issue.key);
      addDetail(details, "Issue URL", getIssueUrl(issue));
      addDetail(details, "Summary", issue.fields?.summary);
      addDetail(details, "Status", issue.fields?.status?.name);
    }

    const updated = listUpdatedFields(context.node.configuration as UpdateIssueConfiguration | undefined);
    if (updated.length > 0) {
      details["Fields Updated"] = updated.join(", ");
    }

    return details;
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const issue = outputs?.default?.[0]?.data as JiraIssue | undefined;
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
  const nodeMetadata = node.metadata as JiraNodeMetadata | undefined;
  const configuration = node.configuration as UpdateIssueConfiguration | undefined;

  addProjectMetadata(metadata, nodeMetadata?.project, configuration?.project);
  addIssueKeyMetadata(metadata, "hash", configuration?.issueKey);

  const updated = listUpdatedFields(configuration);
  if (updated.length > 0) {
    metadata.push({ icon: "edit", label: `Updates: ${updated.join(", ")}` });
  }

  return metadata;
}

function listUpdatedFields(configuration: UpdateIssueConfiguration | undefined): string[] {
  if (!configuration) return [];
  const skip = new Set(["project", "issueKey", "notifyUsers"]);
  const updated: string[] = [];
  (Object.keys(configuration) as (keyof UpdateIssueConfiguration)[]).forEach((key) => {
    if (skip.has(key)) return;
    const value = configuration[key];
    if (value === undefined || value === null) return;
    if (typeof value === "string" && value.trim() === "") return;
    if (Array.isArray(value) && value.length === 0) return;
    updated.push(FIELD_LABELS[key] || key);
  });
  return updated;
}
