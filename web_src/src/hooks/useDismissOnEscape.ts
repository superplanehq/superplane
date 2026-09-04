import { useEffect } from "react";

/**
 * Calls `onDismiss` when the user presses Escape while the document has
 * focus. Skips the callback when the event was already handled elsewhere
 * (`event.defaultPrevented`), so nested controls that use Escape for their
 * own actions (canceling an inline edit, closing a suggestion dropdown, ...)
 * get first refusal. Those handlers must call `preventDefault()` and run
 * before this one, which holds true for React's root-level listener versus
 * this hook's plain `document` listener in the bubble phase.
 *
 * Does nothing when `onDismiss` is not provided.
 */
export function useDismissOnEscape(onDismiss?: () => void): void {
  useEffect(() => {
    if (!onDismiss) {
      return;
    }

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key !== "Escape" || event.defaultPrevented) {
        return;
      }
      onDismiss();
    };

    document.addEventListener("keydown", handleKeyDown);
    return () => {
      document.removeEventListener("keydown", handleKeyDown);
    };
  }, [onDismiss]);
}
