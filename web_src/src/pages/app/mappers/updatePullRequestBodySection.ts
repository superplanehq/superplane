import { noopMapper } from "./noop";
import type { ComponentBaseMapper } from "./types";

interface PublicationMetadata {
  sourceRunId?: string;
  status?: string;
  attempts?: number;
  reason?: string;
  lastError?: string;
  nextRetryAt?: string;
  commentId?: number;
}

// Pull request body sections and managed comments share the publication worker.
export const updatePullRequestBodySectionMapper: ComponentBaseMapper = {
  ...noopMapper,
  getExecutionDetails: ({ execution }) => {
    const publication = (execution.metadata ?? {}) as PublicationMetadata;
    const details: Record<string, string> = {
      "Started At": execution.createdAt ? new Date(execution.createdAt).toLocaleString() : "-",
    };
    const values: Array<[string, string | number | undefined]> = [
      ["Source run ID", publication.sourceRunId],
      ["Status", publication.status],
      ["Attempts", publication.attempts],
      ["Reason", publication.reason],
      ["Last error", publication.lastError],
      ["Next retry", publication.nextRetryAt && new Date(publication.nextRetryAt).toLocaleString()],
      ["Comment ID", publication.commentId || undefined],
    ];
    for (const [label, value] of values) {
      if (value !== undefined && value !== "") details[label] = String(value);
    }
    return details;
  },
};
