import { formatTimestamp } from "../utils";
import type { Issue } from "./types";

// buildAssigneeDetails renders the shared details for the add/remove assignee
// components: the timestamp is always first, followed by the issue, its link,
// the current assignees and state. Kept to at most 6 items.
export function buildAssigneeDetails(issue: Issue, payloadTimestamp?: string): Record<string, string> {
  const details: Record<string, string> = {
    "Updated At": formatTimestamp(issue.updated_at, payloadTimestamp),
    Issue: issue.iid ? `#${issue.iid} ${issue.title || ""}`.trim() : "-",
  };

  addDetailIfPresent(details, "Issue URL", issue.web_url);
  details["Assignees"] = formatAssignees(issue);
  addDetailIfPresent(details, "State", issue.state);

  return details;
}

function formatAssignees(issue: Issue): string {
  const assignees = (issue.assignees ?? [])
    .map((assignee) => (assignee.username ? `@${assignee.username}` : assignee.name))
    .filter(Boolean);

  return assignees.length > 0 ? assignees.join(", ") : "None";
}

function addDetailIfPresent(details: Record<string, string>, label: string, value?: string) {
  if (value) {
    details[label] = value;
  }
}
