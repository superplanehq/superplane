import { useQueryClient } from "@tanstack/react-query";

import type {
  OrganizationsDescribeOrganizationBillingResponse,
  OrganizationsPreviewSeatPurchaseResponse,
  OrganizationsSeatPurchase,
} from "@/api-client";
import { isSeatPurchasePending, validAdditionalSeats } from "@/lib/seatPurchases";
import { useSeatPurchaseBilling } from "./useSeatPurchaseBilling";
import { useSeatPurchaseEstimate } from "./useSeatPurchaseEstimate";
import { useSeatPurchaseInput } from "./useSeatPurchaseInput";
import { useSeatPurchasePolling } from "./useSeatPurchasePolling";
import { useSeatPurchaseState } from "./useSeatPurchaseState";
import { useSeatPurchaseSubmission } from "./useSeatPurchaseSubmission";

export type UseSeatPurchaseResult = {
  isOpen: boolean;
  setOpen: (open: boolean) => void;
  additionalSeats: string;
  setAdditionalSeats: (value: string) => void;
  requestPreview: () => void;
  preview: OrganizationsPreviewSeatPurchaseResponse | undefined;
  previewPending: boolean;
  previewError: string | null;
  submitting: boolean;
  canConfirm: boolean;
  confirm: () => Promise<void>;
  purchase: OrganizationsSeatPurchase | undefined;
  purchaseMessage: string | null;
  /** True after a purchase settled successfully in this session. The dialog hides the form while it holds. */
  attemptSucceeded: boolean;
  /** True when a submission error left the purchase outcome unknown. The UI offers a manual billing refresh. */
  submissionUnresolved: boolean;
  timedOut: boolean;
  refreshBilling: () => Promise<void>;
  refreshing: boolean;
};

export function useSeatPurchase({
  organizationId,
  billing,
  enabled,
  pollingEnabled = enabled,
}: {
  organizationId: string;
  billing: OrganizationsDescribeOrganizationBillingResponse | undefined;
  enabled: boolean;
  pollingEnabled?: boolean;
}): UseSeatPurchaseResult {
  const queryClient = useQueryClient();
  const { lastSeatPurchase, purchasedSeats, purchasableSeats, cancelAtPeriodEnd } = billing ?? {};
  const state = useSeatPurchaseState(organizationId, lastSeatPurchase);
  const input = useSeatPurchaseInput(organizationId, state);
  const count = validAdditionalSeats(input.additionalSeats, purchasableSeats);
  const allowed = enabled && Boolean(organizationId) && !cancelAtPeriodEnd && purchasableSeats !== 0;
  const preview = useSeatPurchaseEstimate({
    organizationId,
    count,
    baseline: purchasedSeats,
    enabled:
      allowed &&
      input.isOpen &&
      count !== undefined &&
      !isSeatPurchasePending(state.purchase) &&
      input.requested &&
      !state.submitting,
    revision: input.revision,
    request: input.request,
    lifecycle: state.lifecycle,
  });
  const recovery = useSeatPurchaseBilling({ organizationId, enabled: pollingEnabled, queryClient, state });
  const confirm = useSeatPurchaseSubmission({
    organizationId,
    queryClient,
    state,
    input,
    preview,
    count,
    baseline: purchasedSeats,
    reloadBilling: recovery.reloadBilling,
  });
  const timedOut = useSeatPurchasePolling({
    organizationId,
    enabled: pollingEnabled,
    queryClient,
    purchase: state.purchase,
    recordPurchase: state.recordPurchase,
  });
  return {
    isOpen: input.isOpen,
    setOpen: input.setOpen,
    additionalSeats: input.additionalSeats,
    setAdditionalSeats: input.setAdditionalSeats,
    requestPreview: input.requestPreview,
    preview: preview.preview,
    previewPending: preview.pending,
    previewError: preview.error,
    submitting: state.submitting,
    canConfirm: preview.canConfirm,
    confirm,
    purchase: state.purchase,
    purchaseMessage: state.message,
    attemptSucceeded: state.attemptSucceeded,
    submissionUnresolved: state.submissionUnresolved,
    timedOut,
    refreshBilling: recovery.refreshBilling,
    refreshing: recovery.refreshing,
  };
}
