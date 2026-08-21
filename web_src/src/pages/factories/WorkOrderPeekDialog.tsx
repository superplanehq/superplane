import { Dialog, DialogContent, DialogDescription, DialogTitle } from "@/components/ui/dialog";

import { WorkOrderDetailPanel } from "./pages/WorkOrderDetailPage";

interface WorkOrderPeekDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  organizationId: string;
  factoryId: string;
  factoryKey: string;
  orderId: string | null;
}

/**
 * Trello-style card overlay. The line board stays behind the dialog.
 */
export function WorkOrderPeekDialog({
  open,
  onOpenChange,
  organizationId,
  factoryId,
  factoryKey,
  orderId,
}: WorkOrderPeekDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        size="large"
        className="flex h-[min(52rem,calc(100vh-3rem))] w-[min(56rem,calc(100vw-2rem))] max-w-none flex-col gap-0 overflow-hidden p-0"
        data-testid="work-order-peek-dialog"
      >
        <DialogTitle className="sr-only">Work order</DialogTitle>
        <DialogDescription className="sr-only">Work order details</DialogDescription>
        <div className="min-h-0 flex-1 overflow-y-auto">
          {orderId ? (
            <WorkOrderDetailPanel
              organizationId={organizationId}
              factoryId={factoryId}
              factoryKey={factoryKey}
              orderId={orderId}
              chrome="dialog"
            />
          ) : null}
        </div>
      </DialogContent>
    </Dialog>
  );
}
