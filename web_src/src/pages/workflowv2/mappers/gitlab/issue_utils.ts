import { Issue } from "./types";

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

  if (issue.closed_by) {
    details["Closed By"] = issue.closed_by.username;
    details["Closed At"] = issue.closed_at ? new Date(issue.closed_at).toLocaleString() : "";
  }

  if (issue.labels && issue.labels.length > 0) {
    details["Labels"] = issue.labels.join(", ");
  }

  if (issue.assignees && issue.assignees.length > 0) {
    details["Assignees"] = issue.assignees.map((assignee) => assignee.username).join(", ");
  }

  if (issue.milestone) {
    details["Milestone"] = issue.milestone.title;
  }

  if (issue.due_date) {
    details["Due Date"] = issue.due_date;
  }

  return details;
}
