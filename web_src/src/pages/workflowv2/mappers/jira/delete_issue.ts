import type React from "react";
import type { ComponentBaseProps } from "@/ui/componentBase";
import type {
  OutputPayload,
  ComponentBaseMapper,
  ComponentBaseContext,
  SubtitleContext,
  ExecutionDetailsContext,
} from "../types";
import { baseProps } from "./base";
import { buildJiraExecutionSubtitle, addDetail } from "./utils";
import type { JiraDeletedIssue } from "./types";

export const deleteIssueMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return baseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const data = outputs?.default?.[0]?.data as JiraDeletedIssue | undefined;
    return buildJiraExecutionSubtitle(context.execution, data?.key ? `${data.key} deleted` : "Issue deleted");
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const data = outputs?.default?.[0]?.data as JiraDeletedIssue | undefined;
    const details: Record<string, string> = {};
    addDetail(details, "Key", data?.key);
    addDetail(details, "ID", data?.id);
    if (data?.deleted) {
      details["Deleted"] = "Yes";
    }
    return details;
  },
};
