import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

import { BILLING_CANCEL_LABEL, BILLING_KEEP_LABEL, billingCancelBusinessConfirmCopy } from "../../lib/billingPlans";

export function BillingCancelBusinessDialog({
  open,
  pending,
  currentPeriodEnd,
  onOpenChange,
  onConfirm,
}: {
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
          <DialogTitle>Cancel Business?</DialogTitle>
          <DialogDescription>{billingCancelBusinessConfirmCopy(currentPeriodEnd)}</DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button
            type="button"
            variant="outline"
            disabled={pending}
            data-testid="billing-cancel-subscription-dismiss"
            onClick={() => onOpenChange(false)}
          >
            {BILLING_KEEP_LABEL}
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
            {pending ? "Canceling Business..." : BILLING_CANCEL_LABEL}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
