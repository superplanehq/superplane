import { useEffect, useRef, type RefObject } from "react";
import type { WorkOrderListState } from "./useWorkOrderListState";

/**
 * Title-bar keyboard shortcuts: `/` opens and focuses search, `F` opens the
 * Filter menu. Both stay out of the way while the user types or holds a
 * modifier, so they never swallow a keystroke meant for the page.
 *
 * Returns the ref to attach to the search input, which the hook focuses
 * whenever the field opens — from the shortcut or from the icon button.
 */
export function useWorkOrdersHeaderShortcuts(state: WorkOrderListState): RefObject<HTMLInputElement | null> {
  const searchRef = useRef<HTMLInputElement>(null);
  const { openSearch, setFilterMenuOpen, searchOpen } = state;

  useEffect(() => {
    function handleShortcut(event: KeyboardEvent) {
      if (event.defaultPrevented || event.metaKey || event.ctrlKey || event.altKey) {
        return;
      }
      const target = event.target as HTMLElement | null;
      if (target && isEditableTarget(target)) {
        return;
      }
      if (event.key === "/") {
        event.preventDefault();
        openSearch();
        return;
      }
      if (event.key === "f" || event.key === "F") {
        event.preventDefault();
        setFilterMenuOpen(true);
      }
    }

    window.addEventListener("keydown", handleShortcut);
    return () => window.removeEventListener("keydown", handleShortcut);
  }, [openSearch, setFilterMenuOpen]);

  useEffect(() => {
    if (searchOpen) {
      searchRef.current?.focus();
    }
  }, [searchOpen]);

  return searchRef;
}

function isEditableTarget(target: HTMLElement): boolean {
  const tag = target.tagName;
  return tag === "INPUT" || tag === "TEXTAREA" || tag === "SELECT" || target.isContentEditable;
}
