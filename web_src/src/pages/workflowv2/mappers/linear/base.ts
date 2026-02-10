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

  details.ID = issue?.id || "-";
  details.Identifier = issue?.identifier || "-";
  details.Title = issue?.title || "-";

  if (issue?.priority !== undefined && issue.priority !== null) {
    details.Priority = priorityLabels[issue.priority] || String(issue.priority);
  }

  if (issue?.team?.id) {
    details["Team ID"] = issue.team.id;
  }

  if (issue?.state?.id) {
    details["State ID"] = issue.state.id;
  }

  if (issue?.assignee?.id) {
    details["Assignee ID"] = issue.assignee.id;
  }

  return details;
}
