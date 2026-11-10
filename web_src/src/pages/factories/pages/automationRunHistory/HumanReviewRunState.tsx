import type { CanvasesCanvasRun } from "@/api-client";
import { Button } from "@/components/ui/button";
import { useWorkOrder } from "@/hooks/useFactoryData";

import { HumanReviewState } from "../HumanReviewState";
import { record } from "./intakeRunSource";

/** Read the current tracked state while the run keeps its immutable source event. */
export function HumanReviewRunState({
  run,
  organizationId,
  factoryId,
}: {
  run: CanvasesCanvasRun;
  organizationId: string;
  factoryId: string;
}) {
  const payload = record(run.rootEvent?.data?.data ?? run.rootEvent?.data);
  const taskId = record(payload.workOrder).id;
  const query = useWorkOrder(organizationId, factoryId, typeof taskId === "string" ? taskId : "");
  const pullRequestId = record(payload.humanReview).pullRequestId;
  const pullRequest = query.data?.pullRequests?.find((item) => item.id === pullRequestId);
  if (query.isLoading)
    return (
      <p role="status" className="mb-4 text-sm text-muted-foreground">
        Loading Request review.
      </p>
    );
  if (query.isError)
    return (
      <div role="alert" className="mb-4 flex items-center gap-3 text-sm">
        <p>Request review could not load.</p>
        <Button variant="outline" size="sm" onClick={() => void query.refetch()}>
          Try again
        </Button>
      </div>
    );
  if (!pullRequest?.humanReview) return null;
  if (pullRequest.humanReview.runId !== run.id)
    return (
      <p className="mb-4 text-sm text-muted-foreground">
        A newer evaluation supersedes this Request review run. Open the task for its current decision.
      </p>
    );
  return (
    <div className="mb-4">
      <HumanReviewState pullRequest={pullRequest} />
    </div>
  );
}
