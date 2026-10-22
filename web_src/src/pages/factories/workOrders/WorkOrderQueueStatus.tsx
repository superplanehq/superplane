import type { FactoriesQueuedTask } from "@/api-client";
import { Button } from "@/components/ui/button";
import { Loader2, X } from "lucide-react";
import type { MouseEvent } from "react";

export function WorkOrderQueueStatus({
  queue,
  orderId,
  title,
  canCancel,
  canDispatch,
  cancellingOrderIds,
  onCancel,
}: {
  queue: FactoriesQueuedTask;
  orderId: string;
  title: string;
  canCancel: boolean;
  canDispatch: boolean;
  cancellingOrderIds?: ReadonlySet<string>;
  onCancel?: (orderId: string, queueId: string) => Promise<void>;
}) {
  const queueId = queue.id;
  const action = queue.name ? `${queue.name} for ${title}` : title;
  const isCancelling = cancellingOrderIds?.has(orderId);
  return (
    <span className="inline-flex shrink-0 items-center gap-1 text-[11px] text-muted-foreground">
      <span role="status">Queued #{queue.position}</span>
      {canCancel && canDispatch && onCancel && queueId ? (
        <Button
          variant="ghost"
          size="icon"
          className="pointer-events-auto size-5 rounded-sm text-muted-foreground hover:text-foreground"
          aria-label={`Remove ${action} from queue`}
          title={queue.name ? `Unschedule ${queue.name}` : "Remove from queue"}
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
