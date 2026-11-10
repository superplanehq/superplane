import { createContext, useContext } from "react";

import { pullRequestChecksKey, type MergeChecksByPullRequest } from "../../../lib/mergeConfidenceScore";
import type { WorkOrderCheckPresentation } from "../../../lib/workOrderChecks";

/**
 * The task summary provides the merge checks of each pull request, so a
 * pull request row shows its own. Without a provider, rows show none.
 */
export const PullRequestChecksContext = createContext<MergeChecksByPullRequest["byRepository"] | null>(null);

export const PullRequestAssessmentFailuresContext = createContext<ReadonlySet<string>>(new Set());

const NO_CHECKS: WorkOrderCheckPresentation[] = [];

export function usePullRequestMergeChecks(repository: string | null | undefined): WorkOrderCheckPresentation[] {
  const byRepository = useContext(PullRequestChecksContext);
  return byRepository?.get(pullRequestChecksKey(repository)) ?? NO_CHECKS;
}
