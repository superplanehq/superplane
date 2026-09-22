import { useRef } from "react";

/** True only after this instance has been pending and then became ready. */
export function useRevealAfterPending(pending: boolean): boolean {
  const sawPending = useRef(pending);
  if (pending) {
    sawPending.current = true;
  }
  return sawPending.current && !pending;
}
