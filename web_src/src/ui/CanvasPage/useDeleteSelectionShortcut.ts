import { useEffect } from "react";

import { isEditableTarget } from "@/lib/editableTarget";

export interface UseDeleteSelectionShortcutOptions {
  /** When true (typically: read-only or live mode), the shortcut is inert. */
  disabled: boolean;
  /** Returns the ids of the nodes currently selected on the canvas. */
  getSelectedNodeIds: () => string[];
  /** Invoked with the selected node ids when a delete shortcut fires. */
  onDelete: (nodeIds: string[]) => void;
}

/**
 * Global keyboard shortcut for deleting the selected canvas component(s).
 *
 * Fires on the plain `Delete` key (Windows Del / macOS forward-delete) and on
 * `Cmd`/`Ctrl`+`Backspace` — the latter keeps working on Mac laptop keyboards
 * that only expose a Backspace-position key, while the modifier requirement
 * avoids clobbering a node when the user is merely deleting characters (see
 * issue #1668). The shortcut is suppressed whenever focus is inside an editable
 * element (input, textarea, contenteditable, or Monaco editor) so it never
 * interferes with typing in configuration fields or code editors.
 */
export function useDeleteSelectionShortcut({
  disabled,
  getSelectedNodeIds,
  onDelete,
}: UseDeleteSelectionShortcutOptions): void {
  useEffect(() => {
    if (disabled) return;

    const handleKeyDown = (event: KeyboardEvent) => {
      const isForwardDelete = event.key === "Delete" && !event.metaKey && !event.ctrlKey && !event.altKey;
      const isModifierBackspace = event.key === "Backspace" && (event.metaKey || event.ctrlKey);
      if (!isForwardDelete && !isModifierBackspace) {
        return;
      }

      if (isEditableTarget(event.target)) {
        return;
      }

      const nodeIds = getSelectedNodeIds();
      if (nodeIds.length === 0) {
        return;
      }

      event.preventDefault();
      onDelete(nodeIds);
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [disabled, getSelectedNodeIds, onDelete]);
}
