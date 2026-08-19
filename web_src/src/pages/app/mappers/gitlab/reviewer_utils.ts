import { formatTimestamp } from "../utils";
import type { MergeRequest } from "./types";

// buildReviewerDetails renders the shared details for the add/remove reviewer
// components: the timestamp is always first, followed by the merge request, its
// link, the current reviewers and state. Kept to at most 6 items.
export function buildReviewerDetails(mergeRequest: MergeRequest, payloadTimestamp?: string): Record<string, string> {
  const details: Record<string, string> = {
    "Updated At": formatTimestamp(mergeRequest.updated_at, payloadTimestamp),
    "Merge Request": mergeRequest.iid ? `!${mergeRequest.iid} ${mergeRequest.title || ""}`.trim() : "-",
  };

  addDetailIfPresent(details, "Merge Request URL", mergeRequest.web_url);
  details["Reviewers"] = formatReviewers(mergeRequest);
  addDetailIfPresent(details, "State", mergeRequest.state);

  return details;
}

function formatReviewers(mergeRequest: MergeRequest): string {
  const reviewers = (mergeRequest.reviewers ?? [])
    .map((reviewer) => (reviewer.username ? `@${reviewer.username}` : reviewer.name))
    .filter(Boolean);

  return reviewers.length > 0 ? reviewers.join(", ") : "None";
}

function addDetailIfPresent(details: Record<string, string>, label: string, value?: string) {
  if (value) {
    details[label] = value;
  }
}
