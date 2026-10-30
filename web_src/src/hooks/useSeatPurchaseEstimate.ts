import { useEffect, useState, type RefObject } from "react";

import { organizationsPreviewSeatPurchase, type OrganizationsPreviewSeatPurchaseResponse } from "@/api-client";
import { getApiErrorMessage } from "@/lib/errors";
import { previewSeatPurchaseBinding } from "@/lib/seatPurchaseOperations";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";

export type CompletedSeatPurchasePreview = {
  estimate: OrganizationsPreviewSeatPurchaseResponse;
  additionalSeats: number;
  billingBaseline: number | undefined;
  /** The billing period end when the preview ran. A renewal makes the preview stale. */
  billingPeriodEnd: string | undefined;
  request: number;
};

export function useSeatPurchaseEstimate({
  organizationId,
  count,
  baseline,
  periodEnd,
  enabled,
  revision,
  request,
  lifecycle,
}: {
  organizationId: string;
  count: number | undefined;
  baseline: number | undefined;
  periodEnd: string | undefined;
  enabled: boolean;
  revision: number;
  request: RefObject<number>;
  lifecycle: RefObject<number>;
}) {
  const [completed, setCompleted] = useState<CompletedSeatPurchasePreview>();
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    setCompleted(undefined);
    setError(null);
    setPending(enabled);
    if (!enabled || count === undefined) return;
    const generation = lifecycle.current;
    let active = true;
    const isCurrent = () => active && revision === request.current && generation === lifecycle.current;
    const timer = window.setTimeout(async () => {
      try {
        const response = await organizationsPreviewSeatPurchase(
          withOrganizationHeader({ organizationId, path: { id: organizationId }, body: { additionalSeats: count } }),
        );
        if (!isCurrent()) return;
        // A preview without its subscription and billing period cannot be confirmed.
        if (response.data?.purchasedSeats === undefined || !previewSeatPurchaseBinding(response.data))
          throw new Error("SuperPlane could not estimate this purchase. Try again.");
        setCompleted({
          estimate: response.data,
          additionalSeats: count,
          billingBaseline: baseline,
          billingPeriodEnd: periodEnd,
          request: revision,
        });
      } catch (previewError) {
        if (isCurrent())
          setError(getApiErrorMessage(previewError, "SuperPlane could not estimate this purchase. Try again."));
      } finally {
        if (isCurrent()) setPending(false);
      }
    }, 300);
    return () => {
      active = false;
      window.clearTimeout(timer);
    };
  }, [organizationId, count, baseline, periodEnd, enabled, revision, request, lifecycle]);

  const current =
    enabled &&
    completed?.request === revision &&
    completed?.billingBaseline === baseline &&
    completed?.billingPeriodEnd === periodEnd
      ? completed
      : undefined;
  return { completed: current, preview: current?.estimate, pending, error, canConfirm: Boolean(current && !pending) };
}
