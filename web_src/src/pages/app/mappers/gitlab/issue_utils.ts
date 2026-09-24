import type { ExecutionInfo } from "../types";
import type { Issue } from "./types";

/**
 * Shared interface for webhook event issue data (object_attributes).
 * This is a subset of the full Issue type used in webhook payloads.
 */
export interface WebhookIssue {
  id?: number;
  iid?: number;
  title?: string;
  description?: string;
  state?: string;
  action?: string;
  url?: string;
}

/**
 * Get display details for a webhook event issue (from object_attributes).
 * Used by trigger renderers.
 */
export function getDetailsForWebhookIssue(issue: WebhookIssue | undefined): Record<string, string> {
  if (!issue) {
    return {};
  }

  return {
    URL: issue.url || "",
    Title: issue.title || "",
    Action: issue.action || "",
    State: issue.state || "",
    IID: issue.iid?.toString() || "",
  };
}

/**
 * Get display details for a full API Issue response.
 * Used by action mappers (create_issue, etc.).
 */
function addDetail(details: Record<string, string>, key: string, value: string | undefined) {
  if (value) {
    details[key] = value;
  }
}

function addClosedDetails(details: Record<string, string>, issue: Issue) {
  if (!issue.closed_by) {
    return;
  }

  details["Closed By"] = issue.closed_by.username;
  details["Closed At"] = issue.closed_at ? new Date(issue.closed_at).toLocaleString() : "";
}

function addJoinedDetail(details: Record<string, string>, key: string, values?: string[]) {
  if (values && values.length > 0) {
    details[key] = values.join(", ");
  }
}

export function getDetailsForApiIssue(issue: Issue | undefined): Record<string, string> {
  if (!issue) {
    return {};
  }

  const details: Record<string, string> = {
    IID: issue.iid?.toString() || "",
    ID: issue.id?.toString() || "",
    State: issue.state || "",
    URL: issue.web_url || "",
    Title: issue.title || "-",
    "Created At": issue.created_at ? new Date(issue.created_at).toLocaleString() : "-",
    "Created By": issue.author?.username || "-",
  };

  addClosedDetails(details, issue);
  addJoinedDetail(details, "Labels", issue.labels);
  addJoinedDetail(
    details,
    "Assignees",
    issue.assignees?.map((assignee) => assignee.username),
  );
  addDetail(details, "Milestone", issue.milestone?.title);
  addDetail(details, "Due Date", issue.due_date);

  return details;
}

/**
 * Get a condensed 6-field execution summary for a full API Issue response.
 * Used by getIssue and updateIssue, whose details panels favor a shorter,
 * at-a-glance view over the full field set in getDetailsForApiIssue.
 */
export function getSummaryDetailsForIssue(execution: ExecutionInfo, issue: Issue | undefined): Record<string, string> {
  if (!issue) {
    return {};
  }

  return {
    "Executed At": execution.createdAt ? new Date(execution.createdAt).toLocaleString() : "-",
    Title: issue.title || "-",
    State: issue.state || "-",
    "Created By": issue.author?.username || "-",
    Labels: issue.labels && issue.labels.length > 0 ? issue.labels.join(", ") : "-",
    URL: issue.web_url || "-",
  };
}
