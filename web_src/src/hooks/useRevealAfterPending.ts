import { useState } from "react";

/** True only after this instance has been pending and then became ready. */
export function useRevealAfterPending(pending: boolean): boolean {
  const [sawPending, setSawPending] = useState(pending);
  if (pending && !sawPending) {
    setSawPending(true);
  }
  return sawPending && !pending;
}
