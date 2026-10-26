import { useCallback } from "react";
import type { QueryClient } from "@tanstack/react-query";

import {
  organizationsPurchaseSeats,
  type OrganizationsDescribeOrganizationBillingResponse,
  type OrganizationsPurchaseSeatsResponse,
  type OrganizationsSeatPurchase,
} from "@/api-client";
import { getApiErrorMessage } from "@/lib/errors";
import {
  confirmSeatPurchaseOperation,
  retireSeatPurchaseOperation,
  type SeatPurchaseOperation,
} from "@/lib/seatPurchaseOperations";
import { validAdditionalSeats } from "@/lib/seatPurchases";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { organizationBillingQueryKey } from "./useOrganizationBilling";
import type { useSeatPurchaseBilling } from "./useSeatPurchaseBilling";
import type { CompletedSeatPurchasePreview, useSeatPurchaseEstimate } from "./useSeatPurchaseEstimate";
import type { useSeatPurchaseInput } from "./useSeatPurchaseInput";
import type { useSeatPurchaseState } from "./useSeatPurchaseState";

type Billing = OrganizationsDescribeOrganizationBillingResponse;
type PurchaseState = ReturnType<typeof useSeatPurchaseState>;
type PreviewInput = ReturnType<typeof useSeatPurchaseInput>;
type Submission = {
  organizationId: string;
  queryClient: QueryClient;
  state: PurchaseState;
  input: PreviewInput;
  generation: number;
  operation: SeatPurchaseOperation;
  reloadBilling: ReturnType<typeof useSeatPurchaseBilling>["reloadBilling"];
};

function errorStatus(error: unknown): number | undefined {
  if (!error || typeof error !== "object") return undefined;
  if ("status" in error && typeof error.status === "number") return error.status;
  if ("response" in error) return errorStatus(error.response);
  return undefined;
}

function billingMatchesPreview(preview: CompletedSeatPurchasePreview, billing: Billing | undefined) {
  if (!billing) return true;
  return (
    billing.purchasedSeats === preview.billingBaseline &&
    billing.currentPeriodEnd === preview.billingPeriodEnd &&
    !billing.cancelAtPeriodEnd &&
    validAdditionalSeats(String(preview.additionalSeats), billing.purchasableSeats) !== undefined
  );
}

function previewIsCurrent({
  preview,
  count,
  baseline,
  revision,
}: {
  preview: CompletedSeatPurchasePreview | undefined;
  count: number | undefined;
  baseline: number | undefined;
  revision: number;
}) {
  return (
    preview !== undefined &&
    preview.additionalSeats === count &&
    preview.billingBaseline === baseline &&
    preview.request === revision
  );
}

function purchaseChanged(
  recovered: OrganizationsSeatPurchase | undefined,
  previous: OrganizationsSeatPurchase | undefined,
) {
  if (!recovered) return false;
  return (
    recovered.id !== previous?.id ||
    recovered.status !== previous?.status ||
    recovered.createdAt !== previous?.createdAt
  );
}

function confirmOperation(organizationId: string, preview: CompletedSeatPurchasePreview, billing: Billing | undefined) {
  const { estimate, additionalSeats } = preview;
  if (estimate.purchasedSeats === undefined) return undefined;
  return confirmSeatPurchaseOperation(
    organizationId,
    { additionalSeats, expectedPurchasedSeats: estimate.purchasedSeats },
    {
      periodStart: estimate.periodStart ?? billing?.currentPeriodStart,
      periodEnd: estimate.periodEnd ?? billing?.currentPeriodEnd,
    },
  );
}

async function submitPurchase(operation: SeatPurchaseOperation) {
  // Direct submission has no automatic retries. Recovery only reads billing.
  const { organizationId, additionalSeats, expectedPurchasedSeats, operationKey } = operation;
  const response = await organizationsPurchaseSeats<false>({
    ...withOrganizationHeader({
      organizationId,
      path: { id: organizationId },
      body: { additionalSeats, expectedPurchasedSeats, operationKey },
    }),
    throwOnError: false,
  });
  if (response.error) {
    throw Object.assign(new Error(getApiErrorMessage(response.error, "SuperPlane could not complete this purchase.")), {
      status: response.response?.status,
    });
  }
  return response.data;
}

function purchaseSettled(purchase: OrganizationsSeatPurchase | undefined) {
  return ["succeeded", "failed", "action_required"].includes(purchase?.status ?? "");
}

function submissionOutcomeUnknown(status: number | undefined) {
  // The server answered definite rejections before charging. A lost response
  // or a server error can have reached Polar, so the outcome stays unknown.
  return status === undefined || status >= 500;
}

async function applyPurchaseResponse(response: OrganizationsPurchaseSeatsResponse | undefined, submission: Submission) {
  const { state, generation, organizationId, queryClient, reloadBilling } = submission;
  // The key stays stored until a server answer for it gives a known outcome.
  // Billing reconciliation does not retire it, because the billing purchase
  // carries no key. The next matching confirmation replays and retires it.
  if (purchaseSettled(response?.purchase)) retireSeatPurchaseOperation(submission.operation);
  if (generation !== state.lifecycle.current) return;
  state.recordPurchase(response?.purchase ?? response?.billing?.lastSeatPurchase);
  if (response?.billing) {
    queryClient.setQueryData(organizationBillingQueryKey(organizationId), response.billing);
    return;
  }
  try {
    const billing = await reloadBilling();
    if (generation === state.lifecycle.current) state.recordPurchase(billing.lastSeatPurchase ?? response?.purchase);
  } catch {
    // A failed billing read must not replace the result that the server confirmed.
  }
}

async function recoverPurchase(error: unknown, submission: Submission) {
  const { state, generation, reloadBilling, input } = submission;
  if (!submissionOutcomeUnknown(errorStatus(error))) retireSeatPurchaseOperation(submission.operation);
  if (generation !== state.lifecycle.current) return;
  const message = getApiErrorMessage(
    error,
    "SuperPlane could not complete this purchase. Refresh billing to check its status.",
  );
  state.setMessage(message);
  let recovered = false;
  try {
    const billing = await reloadBilling();
    if (generation !== state.lifecycle.current) return;
    if (purchaseChanged(billing.lastSeatPurchase, state.purchase)) {
      state.recordPurchase(billing.lastSeatPurchase);
      recovered = true;
    }
  } catch {
    // Preserve the purchase error and expose the explicit billing refresh action.
  }
  if (generation !== state.lifecycle.current) return;
  state.setSubmissionUnresolved(!recovered && submissionOutcomeUnknown(errorStatus(error)));
  if (errorStatus(error) !== 409) return;
  state.setMessage(message);
  input.requestPreview();
}

export function useSeatPurchaseSubmission({
  organizationId,
  queryClient,
  state,
  input,
  preview,
  count,
  baseline,
  reloadBilling,
}: {
  organizationId: string;
  queryClient: QueryClient;
  state: PurchaseState;
  input: PreviewInput;
  preview: ReturnType<typeof useSeatPurchaseEstimate>;
  count: number | undefined;
  baseline: number | undefined;
  reloadBilling: Submission["reloadBilling"];
}) {
  return useCallback(async () => {
    if (state.submissionLock.current || !preview.canConfirm) return;
    const confirmed = preview.completed;
    if (!confirmed || !previewIsCurrent({ preview: confirmed, count, baseline, revision: input.request.current }))
      return;
    const billing = queryClient.getQueryData<Billing>(organizationBillingQueryKey(organizationId));
    if (!billingMatchesPreview(confirmed, billing)) return;
    const operation = confirmOperation(organizationId, confirmed, billing);
    if (!operation) return;
    state.submissionLock.current = true;
    const submission = {
      organizationId,
      queryClient,
      state,
      input,
      reloadBilling,
      operation,
      generation: state.lifecycle.current,
    };
    state.setSubmitting(true);
    input.consumePreview();
    state.clearFeedback();
    try {
      await applyPurchaseResponse(await submitPurchase(operation), submission);
    } catch (error) {
      await recoverPurchase(error, submission);
    } finally {
      if (submission.generation === state.lifecycle.current) {
        state.submissionLock.current = false;
        state.setSubmitting(false);
      }
    }
  }, [organizationId, queryClient, state, input, preview, count, baseline, reloadBilling]);
}
