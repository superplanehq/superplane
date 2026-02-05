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
    return { ...getDetailsForIssue(issue), ...details };
  },
};

function getDetailsForIssue(issue: Issue): Record<string, string> {
  const details: Record<string, string> = {};
  Object.assign(details, {
    "Created At": issue?.created_at ? new Date(issue.created_at).toLocaleString() : "-",
    "Created By": issue?.author?.username || "-",
  });

  details["IID"] = issue?.iid.toString();
  details["ID"] = issue?.id.toString();
  details["State"] = issue?.state;
  details["URL"] = issue?.web_url;
  details["Title"] = issue?.title || "-";

  if (issue.closed_by) {
    details["Closed By"] = issue?.closed_by.username;
    details["Closed At"] = issue?.closed_at ? new Date(issue.closed_at).toLocaleString() : "";
  }

  if (issue.labels && issue.labels.length > 0) {
    details["Labels"] = issue.labels.join(", ");
  }

  if (issue.assignees && issue.assignees.length > 0) {
    details["Assignees"] = issue.assignees.map((assignee) => assignee.username).join(", ");
  }

  return details;
}
