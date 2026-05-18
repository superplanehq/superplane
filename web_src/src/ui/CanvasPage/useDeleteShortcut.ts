import { useEffect } from "react";

export interface UseDeleteShortcutOptions {
  /** When true (read-only, live mode, or no delete handler available), the shortcut is inert. */
  disabled: boolean;
  /** Invoked when a valid delete keystroke fires. Caller handles confirm + delete. */
  onDelete: () => void;
}

/**
 * Global keyboard shortcut for deleting selected canvas components.
 *
 * Fires on plain `Delete` or `Backspace`, no modifiers — between the two,
 * every platform's natural delete keystroke is covered (Windows Delete key,
 * Mac fn+delete, and the Mac key labeled "delete" which sends Backspace).
 *
 * Suppressed whenever focus is in any editable element — plain inputs,
 * textareas, contenteditable regions, or a Monaco editor — so users can
 * still edit text fields without accidentally deleting components (the
 * regression #1668 was filed to fix, which #1717 worked around by
 * disabling the shortcut entirely via `deleteKeyCode={null}`).
 *
 * Restores the keyboard delete UX requested in #3243 without re-introducing
 * the #1668 bug.
 */
export function useDeleteShortcut({ disabled, onDelete }: UseDeleteShortcutOptions): void {
  useEffect(() => {
    if (disabled) return;

    const handleKeyDown = (event: KeyboardEvent) => {
      // Accept plain Delete or Backspace, no modifiers.
      // On macOS the key labeled "delete" sends `Backspace`; the dedicated
      // forward-Delete (fn+delete) sends `Delete`. Between the two, every
      // platform's natural keystroke is covered. Modifier combos are reserved
      // for OS-level text editing shortcuts and not consumed here.
      const isDeleteKey =
        (event.key === "Delete" || event.key === "Backspace") &&
        !event.metaKey &&
        !event.ctrlKey &&
        !event.altKey &&
        !event.shiftKey;
      if (!isDeleteKey) {
        return;
      }

      const target = event.target;
      if (
        target instanceof Element &&
        target.closest('input, textarea, select, [contenteditable="true"], .monaco-editor')
      ) {
        return;
      }

      event.preventDefault();
      onDelete();
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [disabled, onDelete]);
}
