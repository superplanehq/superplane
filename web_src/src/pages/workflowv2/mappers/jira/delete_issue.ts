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
import { addDetail } from "./utils";
import type { DeleteIssueConfiguration, JiraDeletedIssue, JiraNodeMetadata } from "./types";

export const deleteIssueMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return jiraComponentBaseProps(context, metadataList(context.node));
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const details: Record<string, string> = {
      "Executed At": context.execution.createdAt ? new Date(context.execution.createdAt).toLocaleString() : "-",
    };

    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const data = outputs?.default?.[0]?.data as JiraDeletedIssue | undefined;
    addDetail(details, "Key", data?.key);
    addDetail(details, "ID", data?.id);
    if (data?.deleted) {
      details["Status"] = "Deleted";
    }
    return details;
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const data = outputs?.default?.[0]?.data as JiraDeletedIssue | undefined;
    if (data?.key) return `${data.key} deleted`;
    if (context.execution.createdAt) {
      return renderTimeAgo(new Date(context.execution.createdAt));
    }
    return "";
  },
};

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const nodeMetadata = node.metadata as JiraNodeMetadata | undefined;
  const configuration = node.configuration as DeleteIssueConfiguration | undefined;

  const project = nodeMetadata?.project;
  if (project?.name || project?.key) {
    metadata.push({ icon: "folder", label: project?.name || project?.key || "" });
  } else if (configuration?.project) {
    metadata.push({ icon: "folder", label: configuration.project });
  }

  if (configuration?.issueKey && !configuration.issueKey.includes("{{")) {
    metadata.push({ icon: "trash-2", label: configuration.issueKey });
  }

  if (configuration?.deleteSubtasks) {
    metadata.push({ icon: "list-tree", label: "Also subtasks" });
  }

  return metadata;
}
