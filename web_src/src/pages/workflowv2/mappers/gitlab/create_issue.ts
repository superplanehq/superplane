import { ComponentBaseProps } from "@/ui/componentBase";
import {
  OutputPayload,
  ComponentBaseMapper,
  ComponentBaseContext,
  SubtitleContext,
  ExecutionDetailsContext,
} from "../types";
import { Issue } from "./types";
import { baseProps } from "./base";
import { buildGitlabExecutionSubtitle } from "./utils";
import { getDetailsForApiIssue } from "./issue_utils";

export const createIssueMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return baseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);
  },

  subtitle(context: SubtitleContext): string {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    if (outputs?.default?.[0]?.data) {
      const issue = outputs.default[0].data as Issue;
      return `#${issue.iid} ${issue.title}`;
    }
    return buildGitlabExecutionSubtitle(context.execution, "Issue Created");
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const details: Record<string, string> = {};

    if (!outputs || !outputs.default || outputs.default.length === 0) {
      return details;
    }

    if (!outputs.default[0].data) {
      return details;
    }

    const issue = outputs.default[0].data as Issue;
    return { ...getDetailsForApiIssue(issue), ...details };
  },
};
