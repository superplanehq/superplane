import type { FactoriesFactoryPullRequest } from "@/api-client";
import { LoadingButton } from "@/components/ui/loading-button";
import { useExperimentalFeature } from "@/hooks/useExperimentalFeature";
import {
  useFactoryPullRequestMergeability,
  useFixFactoryPullRequestConflicts,
} from "@/hooks/useFactoryPullRequestMerge";
import { FEATURE_FACTORY_PULL_REQUEST_MERGE } from "@/lib/experimentalFeatures";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast } from "@/lib/toast";

import { isActivePRFeedbackActivity } from "../prFeedbackSettingsModel";
import { PULL_REQUEST_REVIEW_COPY, supportsPullRequestMerge } from "./splitRunPullRequestReview";

export function SplitRunPullRequestConflictAction({
  organizationId,
  factoryId,
  orderId,
  pullRequest,
  canAct,
  compact = false,
}: {
  organizationId?: string;
  factoryId?: string;
  orderId?: string;
  pullRequest?: FactoriesFactoryPullRequest;
  canAct: boolean;
  compact?: boolean;
}) {
  const { has } = useExperimentalFeature(organizationId);
  if (!canAct || !pullRequest?.id || !supportsPullRequestMerge(pullRequest)) {
    return null;
  }
  if (!organizationId || !factoryId || !has(FEATURE_FACTORY_PULL_REQUEST_MERGE)) {
    return null;
  }

  return (
    <ConnectedPullRequestConflictAction
      organizationId={organizationId}
      factoryId={factoryId}
      orderId={orderId}
      pullRequest={pullRequest}
      compact={compact}
    />
  );
}

function ConnectedPullRequestConflictAction({
  organizationId,
  factoryId,
  orderId,
  pullRequest,
  compact,
}: {
  organizationId: string;
  factoryId: string;
  orderId?: string;
  pullRequest: FactoriesFactoryPullRequest;
  compact: boolean;
}) {
  const pullRequestId = pullRequest.id ?? "";
  const mergeability = useFactoryPullRequestMergeability(organizationId, factoryId, pullRequestId);
  const fix = useFixFactoryPullRequestConflicts(organizationId, factoryId, orderId);
  const blockedReason = mergeability.data?.blockedReason;
  const feedbackRunning =
    blockedReason === "BLOCKED_REASON_ACTIVE_RUN" ||
    (pullRequest.activities ?? []).some((activity) => isActivePRFeedbackActivity(activity));
  const conflicts =
    blockedReason === "BLOCKED_REASON_CONFLICTING" ||
    (pullRequest.mergeBlockedReason === "CONFLICTING" && feedbackRunning);
  if (!conflicts) {
    return null;
  }

  return (
    <LoadingButton
      type="button"
      size={compact ? "sm" : "lg"}
      variant="outline"
      className={compact ? undefined : "h-11 px-6 text-[15px] font-semibold"}
      loading={fix.isPending}
      loadingText={PULL_REQUEST_REVIEW_COPY.fixingConflicts}
      disabled={feedbackRunning}
      data-testid="split-run-fix-conflicts"
      onClick={() => {
        fix.mutate(pullRequestId, {
          onError: (error) => {
            showErrorToast(getApiErrorMessage(error, "Failed to start the conflict fix."));
          },
        });
      }}
    >
      {PULL_REQUEST_REVIEW_COPY.fixConflicts}
    </LoadingButton>
  );
}
