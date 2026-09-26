import type { FactoriesFactoryPullRequest, FactoriesWorkOrderArtifact } from "@/api-client";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";

import { SEND_WORK_ORDER_TO_BACKLOG_COPY } from "../lib/sendWorkOrderToBacklog";
import { SendWorkOrderToBacklogForm } from "./SendWorkOrderToBacklogForm";

export interface SendWorkOrderToBacklogDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  organizationId: string;
  factoryId: string;
  orderId: string;
  pullRequests?: FactoriesFactoryPullRequest[];
  artifacts?: FactoriesWorkOrderArtifact[];
  canSubmit?: boolean;
}

export function SendWorkOrderToBacklogDialog({
  open,
  onOpenChange,
  organizationId,
  factoryId,
  orderId,
  pullRequests,
  artifacts,
  canSubmit = true,
}: SendWorkOrderToBacklogDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md" data-testid="send-work-order-to-backlog-dialog">
        <DialogHeader>
          <DialogTitle>{SEND_WORK_ORDER_TO_BACKLOG_COPY.action}</DialogTitle>
          <DialogDescription>Move this task to Draft in the Backlog.</DialogDescription>
        </DialogHeader>
        {open ? (
          <SendWorkOrderToBacklogForm
            organizationId={organizationId}
            factoryId={factoryId}
            orderId={orderId}
            pullRequests={pullRequests}
            artifacts={artifacts}
            canSubmit={canSubmit}
            onSent={() => onOpenChange(false)}
          />
        ) : null}
      </DialogContent>
    </Dialog>
  );
}
