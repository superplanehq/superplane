import type { JiraIssue } from "./types";
import { buildSubtitle, buildExecutionSubtitle } from "../utils";

export const buildJiraSubtitle = buildSubtitle;
export const buildJiraExecutionSubtitle = buildExecutionSubtitle;

export function getIssueLabel(issue: JiraIssue | undefined): string {
  if (!issue) {
    return "";
  }
  const summary = issue.fields?.summary;
  if (issue.key && summary) {
    return `${issue.key} · ${summary}`;
  }
  return issue.key || summary || "";
}

export function getProjectLabel(issue: JiraIssue | undefined): string | undefined {
  const project = issue?.fields?.project;
  if (!project) {
    return undefined;
  }
  if (project.name && project.key) {
    return `${project.name} (${project.key})`;
  }
  return project.name || project.key || undefined;
}

export function addDetail(details: Record<string, string>, label: string, value: string | undefined): void {
  if (value && value.trim() !== "") {
    details[label] = value;
  }
}

export function addFormattedTimestamp(
  details: Record<string, string>,
  label: string,
  timestamp: string | undefined,
): void {
  if (!timestamp) {
    return;
  }
  const date = new Date(timestamp);
  if (Number.isNaN(date.getTime())) {
    return;
  }
  details[label] = date.toLocaleString();
}

export function getDetailsForIssue(issue: JiraIssue | undefined): Record<string, string> {
  if (!issue) {
    return {};
  }
  const details: Record<string, string> = {};
  addDetail(details, "Key", issue.key);
  addDetail(details, "Summary", issue.fields?.summary);
  addDetail(details, "Status", issue.fields?.status?.name);
  addDetail(details, "Priority", issue.fields?.priority?.name);
  addDetail(details, "Issue Type", issue.fields?.issuetype?.name);
  addDetail(details, "Assignee", issue.fields?.assignee?.displayName);
  addDetail(details, "Project", getProjectLabel(issue));
  if (issue.fields?.labels && issue.fields.labels.length > 0) {
    details["Labels"] = issue.fields.labels.join(", ");
  }
  addFormattedTimestamp(details, "Created", issue.fields?.created);
  addFormattedTimestamp(details, "Updated", issue.fields?.updated);
  return details;
}
