import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/ui/alertDialog";

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
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent data-testid="billing-cancel-subscription-dialog">
        <AlertDialogHeader>
          <AlertDialogTitle>Cancel Business?</AlertDialogTitle>
          <AlertDialogDescription>{billingCancelBusinessConfirmCopy(currentPeriodEnd)}</AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel data-testid="billing-cancel-subscription-dismiss">{BILLING_KEEP_LABEL}</AlertDialogCancel>
          <AlertDialogAction
            disabled={pending}
            data-testid="billing-cancel-subscription-confirm"
            onClick={(event) => {
              event.preventDefault();
              void (async () => {
                await onConfirm();
                onOpenChange(false);
              })();
            }}
          >
            {pending ? "Canceling Business..." : BILLING_CANCEL_LABEL}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
