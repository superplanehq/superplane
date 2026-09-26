import { useEffect, useId, useState } from "react";

import type { FactoriesFactoryPullRequest, FactoriesWorkOrderArtifact, FactoriesWorkOrderSummary } from "@/api-client";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Label } from "@/components/ui/label";
import { useSendWorkOrderToBacklog, useWorkOrderArtifacts } from "@/hooks/useFactoryData";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast, showSuccessToast } from "@/lib/toast";

import {
  SEND_WORK_ORDER_TO_BACKLOG_COPY,
  workOrderHasClearableArtifacts,
  workOrderHasCloseablePullRequests,
} from "../lib/sendWorkOrderToBacklog";

export interface SendWorkOrderToBacklogFormProps {
  organizationId: string;
  factoryId: string;
  orderId: string;
  pullRequests?: FactoriesFactoryPullRequest[];
  /** Skip the artifacts query when the parent already loaded them. */
  artifacts?: FactoriesWorkOrderArtifact[];
  canSubmit?: boolean;
  onSent?: (order: FactoriesWorkOrderSummary) => void;
}

export function SendWorkOrderToBacklogForm({
  organizationId,
  factoryId,
  orderId,
  pullRequests,
  artifacts,
  canSubmit = true,
  onSent,
}: SendWorkOrderToBacklogFormProps) {
  const closePullRequestsId = useId();
  const clearArtifactsId = useId();
  const [closePullRequests, setClosePullRequests] = useState(false);
  const [clearArtifacts, setClearArtifacts] = useState(false);
  const artifactsQuery = useWorkOrderArtifacts(organizationId, factoryId, orderId);
  const sendToBacklog = useSendWorkOrderToBacklog(organizationId, factoryId);
  const loadedArtifacts = artifacts ?? artifactsQuery.data ?? [];
  const showClosePullRequests = workOrderHasCloseablePullRequests(pullRequests);
  const showClearArtifacts = workOrderHasClearableArtifacts(loadedArtifacts);
  const artifactsPending = artifacts === undefined && artifactsQuery.isLoading;

  useEffect(() => {
    setClosePullRequests(false);
    setClearArtifacts(false);
  }, [orderId]);

  const handleSubmit = async () => {
    try {
      const order = await sendToBacklog.mutateAsync({
        orderId,
        closePullRequests: showClosePullRequests && closePullRequests,
        clearArtifacts: showClearArtifacts && clearArtifacts,
      });
      showSuccessToast(SEND_WORK_ORDER_TO_BACKLOG_COPY.success);
      onSent?.(order);
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, SEND_WORK_ORDER_TO_BACKLOG_COPY.error));
    }
  };

  return (
    <div className="flex flex-col gap-3" data-testid="send-work-order-to-backlog-form">
      {showClosePullRequests ? (
        <div className="flex items-center gap-2">
          <Checkbox
            id={closePullRequestsId}
            checked={closePullRequests}
            onChange={(event) => setClosePullRequests(event.currentTarget.checked)}
            data-testid="send-to-backlog-close-prs"
          />
          <Label htmlFor={closePullRequestsId} className="font-normal">
            {SEND_WORK_ORDER_TO_BACKLOG_COPY.closePullRequests}
          </Label>
        </div>
      ) : null}
      {showClearArtifacts ? (
        <div className="flex items-center gap-2">
          <Checkbox
            id={clearArtifactsId}
            checked={clearArtifacts}
            onChange={(event) => setClearArtifacts(event.currentTarget.checked)}
            data-testid="send-to-backlog-clear-artifacts"
          />
          <Label htmlFor={clearArtifactsId} className="font-normal">
            {SEND_WORK_ORDER_TO_BACKLOG_COPY.clearArtifacts}
          </Label>
        </div>
      ) : null}
      <div>
        <Button
          type="button"
          size="sm"
          disabled={!canSubmit || sendToBacklog.isPending || artifactsPending}
          onClick={() => void handleSubmit()}
          data-testid="send-to-backlog-submit"
        >
          {SEND_WORK_ORDER_TO_BACKLOG_COPY.action}
        </Button>
      </div>
    </div>
  );
}
