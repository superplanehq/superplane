import { Alert, AlertAction, AlertDescription, AlertTitle } from "@/components/reui/alert";
import { Button } from "@/components/ui/button";

import { BILLING_UPDATE_PAYMENT_METHOD_LABEL, billingPaymentFailedCopy } from "../../lib/billingPlans";

type BillingPaymentFailedBannerProps = {
  failedAt?: string;
  accessEndsAt?: string;
  canManageBilling: boolean;
  pending: boolean;
  onUpdatePaymentMethod: () => void;
};

/** Tells the owner that the renewal failed, what stops, and how to recover. */
export function BillingPaymentFailedBanner({
  failedAt,
  accessEndsAt,
  canManageBilling,
  pending,
  onUpdatePaymentMethod,
}: BillingPaymentFailedBannerProps) {
  return (
    <Alert variant="warning" data-testid="billing-payment-failed">
      <AlertTitle>Payment failed</AlertTitle>
      <AlertDescription>
        <p>{billingPaymentFailedCopy(failedAt, accessEndsAt)}</p>
      </AlertDescription>
      {canManageBilling ? (
        <AlertAction>
          <Button type="button" size="sm" disabled={pending} onClick={onUpdatePaymentMethod}>
            {BILLING_UPDATE_PAYMENT_METHOD_LABEL}
          </Button>
        </AlertAction>
      ) : null}
    </Alert>
  );
}
