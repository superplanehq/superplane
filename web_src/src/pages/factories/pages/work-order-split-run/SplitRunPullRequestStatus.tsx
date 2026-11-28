import type { FactoriesFactoryPullRequest } from "@/api-client";
import { useExperimentalFeature } from "@/hooks/useExperimentalFeature";
import { useFactoryPullRequestMergeability } from "@/hooks/useFactoryPullRequestMergeability";
import { FEATURE_FACTORY_PULL_REQUEST_MERGE } from "@/lib/experimentalFeatures";

import { isMergedPullRequest, PULL_REQUEST_REVIEW_COPY, supportsPullRequestStatus } from "./splitRunPullRequestReview";

export function SplitRunPullRequestStatus({
  organizationId,
  factoryId,
  pullRequest,
  compact = false,
}: {
  organizationId?: string;
  factoryId?: string;
  pullRequest?: FactoriesFactoryPullRequest;
  compact?: boolean;
}) {
  const { has } = useExperimentalFeature(organizationId);
  const supported = supportsPullRequestStatus(pullRequest) && Boolean(pullRequest?.id);
  const merged = supported && isMergedPullRequest(pullRequest);
  const enabled = supported && !merged && has(FEATURE_FACTORY_PULL_REQUEST_MERGE);
  const mergeability = useFactoryPullRequestMergeability(organizationId ?? "", factoryId ?? "", pullRequest?.id ?? "", {
    enabled,
  });
  let message = enabled && !mergeability.data?.canMerge ? mergeability.data?.message : undefined;
  if (merged) {
    message = PULL_REQUEST_REVIEW_COPY.merged;
  }
  if (!message) {
    return null;
  }
  return (
    <p
      className={compact ? "text-[12px] leading-4 text-foreground/70" : "text-[13px] leading-5 text-foreground/70"}
      data-testid={merged ? "split-run-pr-merged" : "split-run-pr-status"}
    >
      {message}
    </p>
  );
}
