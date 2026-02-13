import { Issue } from "./types";

const priorityLabels: Record<number, string> = {
  0: "None",
  1: "Urgent",
  2: "High",
  3: "Medium",
  4: "Low",
};

export function getDetailsForIssue(issue: Issue): Record<string, string> {
  const details: Record<string, string> = {};

  Object.assign(details, {
    "Created At": issue?.createdAt ? new Date(issue.createdAt).toLocaleString() : "-",
  });

  details.Identifier = issue?.identifier || "-";
  details.Title = issue?.title || "-";

  if (issue?.priority !== undefined && issue.priority !== null) {
    details.Priority = priorityLabels[issue.priority] || String(issue.priority);
  }

  if (issue?.url) {
    details.URL = issue.url;
  }

  return details;
}
