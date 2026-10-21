import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

import { BILLING_BUSINESS_NAME, billingCancelBusinessConfirmCopy } from "../../lib/billingPlans";

export function BillingCancelBusinessDialog({
  planName = BILLING_BUSINESS_NAME,
  open,
  pending,
  currentPeriodEnd,
  onOpenChange,
  onConfirm,
}: {
  planName?: string;
  open: boolean;
  pending: boolean;
  currentPeriodEnd?: string;
  onOpenChange: (open: boolean) => void;
  onConfirm: () => void | Promise<void>;
}) {
  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (pending && !next) {
          return;
        }
        onOpenChange(next);
      }}
    >
      <DialogContent data-testid="billing-cancel-subscription-dialog">
        <DialogHeader>
          <DialogTitle>Cancel {planName}?</DialogTitle>
          <DialogDescription>{billingCancelBusinessConfirmCopy(currentPeriodEnd, planName)}</DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button
            type="button"
            variant="outline"
            disabled={pending}
            data-testid="billing-cancel-subscription-dismiss"
            onClick={() => onOpenChange(false)}
          >
            {`Keep ${planName}`}
          </Button>
          <Button
            type="button"
            disabled={pending}
            data-testid="billing-cancel-subscription-confirm"
            onClick={() => {
              void (async () => {
                try {
                  await onConfirm();
                  onOpenChange(false);
                } catch {
                  // Keep the dialog open so the owner can retry.
                }
              })();
            }}
          >
            {pending ? `Canceling ${planName}...` : `Cancel ${planName}`}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
