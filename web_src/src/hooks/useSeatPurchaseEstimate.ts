import { useEffect, useState, type RefObject } from "react";

import { organizationsPreviewSeatPurchase, type OrganizationsPreviewSeatPurchaseResponse } from "@/api-client";
import { getApiErrorMessage } from "@/lib/errors";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";

export type CompletedSeatPurchasePreview = {
  estimate: OrganizationsPreviewSeatPurchaseResponse;
  additionalSeats: number;
  billingBaseline: number | undefined;
  request: number;
  /** Identifies the purchase of this preview. A retry of the same preview sends the same key, so the server cannot charge twice. */
  operationKey: string;
};

export function useSeatPurchaseEstimate({
  organizationId,
  count,
  baseline,
  enabled,
  revision,
  request,
  lifecycle,
}: {
  organizationId: string;
  count: number | undefined;
  baseline: number | undefined;
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
        if (response.data?.purchasedSeats === undefined)
          throw new Error("SuperPlane could not estimate this purchase. Try again.");
        setCompleted({
          estimate: response.data,
          additionalSeats: count,
          billingBaseline: baseline,
          request: revision,
          operationKey: crypto.randomUUID(),
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
  }, [organizationId, count, baseline, enabled, revision, request, lifecycle]);

  const current =
    enabled && completed?.request === revision && completed?.billingBaseline === baseline ? completed : undefined;
  return { completed: current, preview: current?.estimate, pending, error, canConfirm: Boolean(current && !pending) };
}
