import { useEffect, useRef } from "react";

/**
 * Tracks whether a field was shown read-only (live configuration view) so default
 * application effects can skip firing when the form becomes editable again.
 */
export function useSkipDefaultsAfterReadOnly(readOnly: boolean, contextKey?: string): boolean {
  const wasReadOnlyRef = useRef(false);
  const contextKeyRef = useRef(contextKey);

  if (contextKey !== undefined && contextKeyRef.current !== contextKey) {
    wasReadOnlyRef.current = false;
    contextKeyRef.current = contextKey;
  }

  useEffect(() => {
    if (readOnly) {
      wasReadOnlyRef.current = true;
    }
  }, [readOnly]);

  return wasReadOnlyRef.current;
}
