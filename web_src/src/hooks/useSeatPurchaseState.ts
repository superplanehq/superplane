import { useCallback, useEffect, useRef, useState } from "react";

import type { OrganizationsSeatPurchase } from "@/api-client";
import { recentSeatPurchase, seatPurchaseMessage } from "@/lib/seatPurchases";

export function useSeatPurchaseState(organizationId: string, billingPurchase: OrganizationsSeatPurchase | undefined) {
  const [purchaseState, setPurchaseState] = useState({ organizationId, purchase: billingPurchase });
  const [message, setMessage] = useState<string | null>(seatPurchaseMessage(recentSeatPurchase(billingPurchase)));
  const [submitting, setSubmitting] = useState(false);
  const lifecycle = useRef(0);
  const submissionLock = useRef(false);

  useEffect(() => {
    lifecycle.current += 1;
    setPurchaseState({ organizationId, purchase: undefined });
    setMessage(null);
    setSubmitting(false);
    submissionLock.current = false;
    return () => {
      lifecycle.current += 1;
    };
  }, [organizationId]);

  useEffect(() => {
    if (!billingPurchase) return;
    setPurchaseState({ organizationId, purchase: billingPurchase });
    setMessage(seatPurchaseMessage(recentSeatPurchase(billingPurchase)));
  }, [organizationId, billingPurchase]);

  const recordPurchase = useCallback(
    (purchase: OrganizationsSeatPurchase | undefined) => {
      if (!purchase) return;
      setPurchaseState({ organizationId, purchase });
      setMessage(seatPurchaseMessage(recentSeatPurchase(purchase)));
    },
    [organizationId],
  );

  return {
    purchase: purchaseState.organizationId === organizationId ? purchaseState.purchase : billingPurchase,
    message,
    setMessage,
    recordPurchase,
    submitting,
    setSubmitting,
    submissionLock,
    lifecycle,
  };
}
