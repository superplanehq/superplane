import { useCallback, useEffect, useRef, useState } from "react";

import { isSeatPurchasePending, seatPurchaseMessage } from "@/lib/seatPurchases";
import type { useSeatPurchaseState } from "./useSeatPurchaseState";

export function useSeatPurchaseInput(organizationId: string, state: ReturnType<typeof useSeatPurchaseState>) {
  const [isOpen, setIsOpen] = useState(false);
  const [additionalSeats, setQuantity] = useState("1");
  const [requested, setRequested] = useState(false);
  const [revision, setRevision] = useState(0);
  const request = useRef(0);
  const dialogResetCount = useRef(0);
  const { submissionLock, purchase, setMessage, clearFeedback } = state;
  const pending = isSeatPurchasePending(purchase);

  const invalidate = useCallback(() => {
    request.current += 1;
    setRevision(request.current);
  }, []);

  useEffect(() => {
    setIsOpen(false);
    setQuantity("1");
    setRequested(false);
    invalidate();
    dialogResetCount.current += 1;
  }, [organizationId, invalidate]);

  const requestPreview = useCallback(() => {
    setRequested(true);
    invalidate();
  }, [invalidate]);

  const consumePreview = useCallback(() => {
    setRequested(false);
    invalidate();
  }, [invalidate]);

  const setOpen = useCallback(
    (open: boolean) => {
      if (open === isOpen) return;
      invalidate();
      if (open && !submissionLock.current) {
        setQuantity("1");
        setRequested(true);
        clearFeedback();
        if (pending) setMessage(seatPurchaseMessage(purchase));
      }
      setIsOpen(open);
    },
    [isOpen, invalidate, submissionLock, pending, purchase, setMessage, clearFeedback],
  );

  const setAdditionalSeats = useCallback(
    (value: string) => {
      if (submissionLock.current || pending) return;
      setQuantity(value);
      requestPreview();
      clearFeedback();
    },
    [submissionLock, pending, requestPreview, clearFeedback],
  );

  return {
    isOpen,
    setOpen,
    additionalSeats,
    setAdditionalSeats,
    requested,
    revision,
    request,
    requestPreview,
    consumePreview,
    dialogResetCount,
  };
}
