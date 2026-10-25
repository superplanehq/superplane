import { useCallback, useEffect, useRef, useState } from "react";

import type { OrganizationsSeatPurchase } from "@/api-client";
import { loadSeatPurchaseOperations } from "@/lib/seatPurchaseOperations";
import { recentSeatPurchase, seatPurchaseMessage } from "@/lib/seatPurchases";

export function useSeatPurchaseState(organizationId: string, billingPurchase: OrganizationsSeatPurchase | undefined) {
  const [purchaseState, setPurchaseState] = useState({ organizationId, purchase: billingPurchase });
  const [message, setMessage] = useState<string | null>(seatPurchaseMessage(recentSeatPurchase(billingPurchase)));
  const [submitting, setSubmitting] = useState(false);
  // attemptSucceeded hides the form only for a purchase settled in this
  // session, so an earlier success cannot block a new attempt.
  const [attemptSucceeded, setAttemptSucceeded] = useState(false);
  // submissionUnresolved offers the manual billing refresh when a submission
  // error leaves the purchase outcome unknown.
  const [submissionUnresolved, setSubmissionUnresolved] = useState(false);
  const lifecycle = useRef(0);
  const submissionLock = useRef(false);

  const clearFeedback = useCallback(() => {
    setMessage(null);
    setAttemptSucceeded(false);
    setSubmissionUnresolved(false);
  }, []);

  useEffect(() => {
    lifecycle.current += 1;
    setPurchaseState({ organizationId, purchase: undefined });
    setSubmitting(false);
    clearFeedback();
    // An operation stored before a page refresh has an unknown outcome. Offer
    // the billing refresh until billing shows a purchase status.
    setSubmissionUnresolved(loadSeatPurchaseOperations(organizationId).length > 0);
    submissionLock.current = false;
    return () => {
      lifecycle.current += 1;
    };
  }, [organizationId, clearFeedback]);

  useEffect(() => {
    if (!billingPurchase) return;
    setPurchaseState({ organizationId, purchase: billingPurchase });
    setMessage(seatPurchaseMessage(recentSeatPurchase(billingPurchase)));
    setSubmissionUnresolved(false);
  }, [organizationId, billingPurchase]);

  const recordPurchase = useCallback(
    (purchase: OrganizationsSeatPurchase | undefined) => {
      if (!purchase) return;
      setPurchaseState({ organizationId, purchase });
      setMessage(seatPurchaseMessage(recentSeatPurchase(purchase)));
      setSubmissionUnresolved(false);
      if (purchase.status === "succeeded") setAttemptSucceeded(true);
    },
    [organizationId],
  );

  return {
    purchase: purchaseState.organizationId === organizationId ? purchaseState.purchase : billingPurchase,
    message,
    setMessage,
    recordPurchase,
    clearFeedback,
    submitting,
    setSubmitting,
    attemptSucceeded,
    submissionUnresolved,
    setSubmissionUnresolved,
    submissionLock,
    lifecycle,
  };
}
