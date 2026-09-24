import type React from "react";
import type { ComponentBaseProps } from "@/ui/componentBase";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  OutputPayload,
  SubtitleContext,
} from "../types";
import { baseProps } from "./base";
import { buildGithubExecutionSubtitle } from "./utils";

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
  props(context: ComponentBaseContext): ComponentBaseProps {
    return baseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);
  },
  subtitle(context: SubtitleContext): string | React.ReactNode {
    return buildGithubExecutionSubtitle(context.execution);
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;

    if (!outputs?.default?.length) {
      return {};
    }

    return detailsFromCommitStatus(outputs.default[0].data as CommitStatus);
  },
};

function detailsFromCommitStatus(status: CommitStatus): Record<string, string> {
  const details = commitStatusTimestamps(status);
  Object.assign(details, commitStatusFields(status));
  return details;
}

function commitStatusTimestamps(status: CommitStatus): Record<string, string> {
  const details: Record<string, string> = {
    "Created At": status.created_at ? new Date(status.created_at).toLocaleString() : "-",
    "Created By": status.creator?.login || "-",
  };

  if (status.updated_at) {
    details["Updated At"] = new Date(status.updated_at).toLocaleString();
  }

  return details;
}

function commitStatusFields(status: CommitStatus): Record<string, string> {
  return {
    "Commit Status": status?.state || "",
    Context: status?.context || "",
    Description: status?.description || "",
    "Target URL": status?.target_url || "",
    "Status ID": status?.id?.toString() || "",
  };
}
