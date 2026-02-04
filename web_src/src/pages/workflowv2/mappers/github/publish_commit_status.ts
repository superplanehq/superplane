import { ComponentBaseProps } from "@/ui/componentBase";
import {
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
  subtitle(context: SubtitleContext): string {
    return buildGithubExecutionSubtitle(context.execution);
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    let details: Record<string, string> = {};

    if (outputs && outputs.default && outputs.default.length > 0) {
      const status = outputs.default[0].data as CommitStatus;
      Object.assign(details, {
        "Created At": status.created_at ? new Date(status.created_at).toLocaleString() : "-",
        "Created By": status.creator?.login || "-",
      });

      if (status.updated_at) {
        details["Updated At"] = new Date(status.updated_at).toLocaleString();
      }

      details["Commit Status"] = status?.state || "";
      details["Context"] = status?.context || "";
      details["Description"] = status?.description || "";
      details["Target URL"] = status?.target_url || "";
      details["Status ID"] = status?.id?.toString() || "";
    }

    return details;
  },
};
