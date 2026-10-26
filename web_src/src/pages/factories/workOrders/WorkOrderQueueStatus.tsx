import type { FactoriesQueuedRunnerJob } from "@/api-client";
import { Button } from "@/components/ui/button";
import { Loader2, X } from "lucide-react";
import type { MouseEvent } from "react";

type WorkOrderQueueActions = {
  orderId: string;
  title: string;
  canCancel: boolean;
  canDispatch: boolean;
  cancellingJobIds?: ReadonlySet<string>;
  onCancel?: (orderId: string, queueId: string) => Promise<void>;
};

export function WorkOrderQueueSummary({
  queues,
  automationQueued,
  ...actions
}: WorkOrderQueueActions & {
  queues: FactoriesQueuedRunnerJob[];
  automationQueued: boolean;
}) {
  if (!automationQueued && queues.length === 0) return null;
  return (
    <span className="flex flex-wrap items-center justify-end gap-1.5">
      {automationQueued ? (
        <span role="status" className="text-[11px] text-muted-foreground">
          Waiting for automation slot
        </span>
      ) : null}
      {queues.map((queue) => (
        <WorkOrderQueueStatus key={queue.id} queue={queue} {...actions} />
      ))}
    </span>
  );
}

export function WorkOrderQueueStatus({
  queue,
  orderId,
  title,
  canCancel,
  canDispatch,
  cancellingJobIds,
  onCancel,
}: WorkOrderQueueActions & {
  queue: FactoriesQueuedRunnerJob;
}) {
  const queueId = queue.id;
  const jobName = queue.nodeName || queue.automationName;
  const action = jobName ? `${jobName} runner job for ${title}` : `runner job for ${title}`;
  const isCancelling = Boolean(queueId && cancellingJobIds?.has(queueId));
  return (
    <span className="inline-flex shrink-0 items-center gap-1 text-[11px] text-muted-foreground">
      <span role="status" title="Organization runner queue. Available fleets can start later jobs first.">
        {queue.nodeName || queue.automationName ? `${queue.nodeName || queue.automationName}: ` : ""}Runner queued #
        {queue.position}
      </span>
      {canCancel && canDispatch && onCancel && queueId ? (
        <Button
          variant="ghost"
          size="icon"
          className="pointer-events-auto size-5 rounded-sm text-muted-foreground hover:text-foreground"
          aria-label={`Cancel ${action}`}
          title={queue.automationName ? `Cancel ${queue.automationName} runner job` : "Cancel runner job"}
          disabled={isCancelling}
          onClick={(event: MouseEvent<HTMLButtonElement>) => {
            event.preventDefault();
            event.stopPropagation();
            void onCancel(orderId, queueId);
          }}
        >
          {isCancelling ? (
            <Loader2 className="size-3 animate-spin" aria-hidden />
          ) : (
            <X className="size-3" aria-hidden />
          )}
        </Button>
      ) : null}
    </span>
  );
}
