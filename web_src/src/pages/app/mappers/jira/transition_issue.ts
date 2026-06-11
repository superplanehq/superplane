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
import type { JiraIssue, JiraNodeMetadata, TransitionIssueConfiguration } from "./types";

export const transitionIssueMapper: ComponentBaseMapper = {
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

    const configuration = context.node.configuration as TransitionIssueConfiguration | undefined;
    addDetail(details, "Target Status", configuration?.targetStatus);
    addDetail(details, "Resolution", configuration?.resolution);

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
  const configuration = node.configuration as TransitionIssueConfiguration | undefined;

  addProjectMetadata(metadata, nodeMetadata?.project, configuration?.project);
  addIssueKeyMetadata(metadata, "hash", configuration?.issueKey);

  const status = nodeMetadata?.status || configuration?.targetStatus;
  if (status) {
    metadata.push({ icon: "flag", label: status });
  }

  if (configuration?.resolution) {
    metadata.push({ icon: "circle-check", label: configuration.resolution });
  }

  return metadata;
}
