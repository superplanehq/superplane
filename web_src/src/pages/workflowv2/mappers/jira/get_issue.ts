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
import { addDetail, getIssueLabel } from "./utils";
import type { GetIssueConfiguration, JiraIssue, JiraNodeMetadata } from "./types";

export const getIssueMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return jiraComponentBaseProps(context, metadataList(context.node));
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const details: Record<string, string> = {
      "Executed At": context.execution.createdAt ? new Date(context.execution.createdAt).toLocaleString() : "-",
    };

    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const issue = outputs?.default?.[0]?.data as JiraIssue | undefined;
    if (!issue) return details;

    addDetail(details, "Key", issue.key);
    addDetail(details, "Summary", issue.fields?.summary);
    addDetail(details, "Status", issue.fields?.status?.name);
    addDetail(details, "Assignee", issue.fields?.assignee?.displayName);
    addDetail(details, "Priority", issue.fields?.priority?.name);

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
  const configuration = node.configuration as GetIssueConfiguration | undefined;

  const project = nodeMetadata?.project;
  if (project?.name || project?.key) {
    metadata.push({ icon: "folder", label: project?.name || project?.key || "" });
  } else if (configuration?.project) {
    metadata.push({ icon: "folder", label: configuration.project });
  }

  if (configuration?.issueKey && !configuration.issueKey.includes("{{")) {
    metadata.push({ icon: "hash", label: configuration.issueKey });
  }

  return metadata;
}
