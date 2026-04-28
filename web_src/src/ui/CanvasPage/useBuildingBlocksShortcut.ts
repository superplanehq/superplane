import { useEffect } from "react";

export interface UseBuildingBlocksShortcutOptions {
  /** When true (typically: read-only, live mode, embedded preview), the shortcut is inert. */
  disabled: boolean;
  /** The sidebar is already open — the shortcut matches the `+` button's availability, so we do nothing. */
  isSidebarOpen: boolean;
  /** Invoked when the user presses `c` in a context where the sidebar should open. */
  onOpen: () => void;
}

/**
 * Global `c` shortcut for opening the add-component panel.
 *
 * Behaves as a keyboard equivalent of clicking the `+` control: active in the
 * same modes the `+` button is visible, ignored while the panel is already up
 * (the `+` button is hidden then too), and suppressed whenever the user is
 * typing in any editable element — plain inputs, textareas, contenteditable
 * regions, or a Monaco editor — so the shortcut never eats text input.
 */
export function useBuildingBlocksShortcut({ disabled, isSidebarOpen, onOpen }: UseBuildingBlocksShortcutOptions): void {
  useEffect(() => {
    if (disabled) return;

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key !== "c" || event.metaKey || event.ctrlKey || event.altKey) {
        return;
      }

      const target = event.target;
      if (
        target instanceof Element &&
        target.closest('input, textarea, select, [contenteditable="true"], .monaco-editor')
      ) {
        return;
      }

      if (isSidebarOpen) {
        return;
      }

      event.preventDefault();
      onOpen();
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [disabled, isSidebarOpen, onOpen]);
}
