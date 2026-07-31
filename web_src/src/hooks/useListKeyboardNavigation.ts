import { useCallback, useEffect, useState } from "react";
import type { KeyboardEvent } from "react";

interface UseListKeyboardNavigationOptions {
  /** Number of items currently rendered in the list. */
  itemCount: number;
  /** Called with the highlighted index when the user presses Enter. */
  onActivate?: (index: number) => void;
}

interface ListKeyboardNavigation {
  /** Index of the highlighted item, or -1 when the list is empty. */
  activeIndex: number;
  setActiveIndex: (index: number) => void;
  handleKeyDown: (event: KeyboardEvent<HTMLElement>) => void;
}

const NO_ACTIVE_INDEX = -1;

/**
 * Drives a highlighted row from a search input: Arrow keys move the highlight,
 * Home/End jump to the ends, and Enter activates the highlighted item. Keeps
 * focus where it is so a filter input stays typable while navigating.
 *
 * Events carrying a modifier are ignored so application-level shortcuts
 * (e.g. the global command palette on Cmd+/) keep working.
 */
export function useListKeyboardNavigation({
  itemCount,
  onActivate,
}: UseListKeyboardNavigationOptions): ListKeyboardNavigation {
  const [activeIndex, setActiveIndex] = useState(itemCount > 0 ? 0 : NO_ACTIVE_INDEX);

  useEffect(() => {
    setActiveIndex((current) => {
      if (itemCount === 0) return NO_ACTIVE_INDEX;
      if (current < 0) return 0;
      return Math.min(current, itemCount - 1);
    });
  }, [itemCount]);

  const handleKeyDown = useCallback(
    (event: KeyboardEvent<HTMLElement>) => {
      if (event.metaKey || event.ctrlKey || event.altKey) return;
      if (itemCount === 0) return;

      switch (event.key) {
        case "ArrowDown":
          event.preventDefault();
          setActiveIndex((current) => (current + 1) % itemCount);
          return;
        case "ArrowUp":
          event.preventDefault();
          setActiveIndex((current) => (current - 1 + itemCount) % itemCount);
          return;
        case "Home":
          event.preventDefault();
          setActiveIndex(0);
          return;
        case "End":
          event.preventDefault();
          setActiveIndex(itemCount - 1);
          return;
        case "Enter":
          if (activeIndex < 0 || !onActivate) return;
          event.preventDefault();
          onActivate(activeIndex);
          return;
        default:
          return;
      }
    },
    [activeIndex, itemCount, onActivate],
  );

  return { activeIndex, setActiveIndex, handleKeyDown };
}
