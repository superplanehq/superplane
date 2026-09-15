import { useRef } from "react";

import { stabilizeItemOrder } from "@/pages/factories/lib/stabilizeItemOrder";

/**
 * Pins items to the last rendered order until `resetKey` changes.
 * Queue and cancel bump updatedAt; this keeps those cards in place.
 */
export function useStabilizedItems<T>(items: T[], idOf: (item: T) => string | undefined, resetKey: string): T[] {
  const previousIdsRef = useRef<string[]>([]);
  const resetKeyRef = useRef(resetKey);

  if (resetKeyRef.current !== resetKey) {
    previousIdsRef.current = [];
    resetKeyRef.current = resetKey;
  }

  const next = stabilizeItemOrder(previousIdsRef.current, items, idOf);
  previousIdsRef.current = next.flatMap((item) => {
    const id = idOf(item);
    return id ? [id] : [];
  });
  return next;
}
