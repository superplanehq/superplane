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
import { buildJiraExecutionSubtitle, getDetailsForIssue, getIssueLabel } from "./utils";
import type { JiraIssue } from "./types";

export const getIssueMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return baseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const issue = outputs?.default?.[0]?.data as JiraIssue | undefined;
    return buildJiraExecutionSubtitle(context.execution, getIssueLabel(issue));
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const issue = outputs?.default?.[0]?.data as JiraIssue | undefined;
    return getDetailsForIssue(issue);
  },
};
