import { noopMapper } from "./noop";
import type { ComponentBaseMapper } from "./types";

interface PublicationMetadata {
  sourceRunId?: string;
  status?: string;
  attempts?: number;
  reason?: string;
  lastError?: string;
  nextRetryAt?: string;
}

export const updatePullRequestBodySectionMapper: ComponentBaseMapper = {
  ...noopMapper,
  getExecutionDetails: ({ execution }) => {
    const publication = execution.metadata as PublicationMetadata | undefined;
    const details: Record<string, string> = {
      "Started At": execution.createdAt ? new Date(execution.createdAt).toLocaleString() : "-",
    };
    if (publication?.sourceRunId) details["Source run ID"] = publication.sourceRunId;
    if (publication?.status) details.Status = publication.status;
    if (publication?.attempts !== undefined) details.Attempts = String(publication.attempts);
    if (publication?.reason) details.Reason = publication.reason;
    if (publication?.lastError) details["Last error"] = publication.lastError;
    if (publication?.nextRetryAt) details["Next retry"] = new Date(publication.nextRetryAt).toLocaleString();
    return details;
  },
};
