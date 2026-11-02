import { Link } from "@/components/Link/link";
import { useBillingProductEnabled } from "@/hooks/useBillingProductEnabled";
import { getApiErrorMessage } from "@/lib/errors";
import { organizationSettingsSectionPath } from "../lib/factoryPagePaths";
import { workOrderCreditFailureFromMessage } from "../lib/workOrderFailureReason";

export function CodexSetupFailure({ error, organizationId }: { error: unknown; organizationId: string }) {
  const billingEnabled = useBillingProductEnabled();
  if (!error) return null;
  const message = getApiErrorMessage(error, "SuperPlane could not complete ChatGPT setup. Try again.");
  const creditFailure = workOrderCreditFailureFromMessage(message);
  const actionLabel =
    creditFailure?.actionLabel ??
    (message === "Add a payment method before you complete workspace setup." ? "View billing" : undefined);
  return (
    <p role="alert" className="text-xs text-muted-foreground">
      {creditFailure?.message ?? message}
      {billingEnabled && actionLabel ? (
        <>
          {" "}
          <Link
            href={organizationSettingsSectionPath(organizationId, "billing")}
            className="font-medium text-foreground underline underline-offset-2"
          >
            {actionLabel}
          </Link>
        </>
      ) : null}
    </p>
  );
}
