import { useCallback, useState } from "react";

export const WORK_ORDER_FULL_PAGE_STORAGE_KEY = "sp:work-order:full-page";

/** Read the task-modal full-page preference. Missing or invalid values stay compact. */
export function readStoredWorkOrderFullPage(): boolean {
  try {
    return window.localStorage.getItem(WORK_ORDER_FULL_PAGE_STORAGE_KEY) === "1";
  } catch {
    return false;
  }
}

function persistWorkOrderFullPage(fullPage: boolean): void {
  try {
    window.localStorage.setItem(WORK_ORDER_FULL_PAGE_STORAGE_KEY, fullPage ? "1" : "0");
  } catch {
    // Ignore storage failures (private mode, quota, etc.).
  }
}

/** Owns the task-modal full-page toggle across opens. */
export function useWorkOrderFullPagePreference(): {
  fullPage: boolean;
  toggleFullPage: () => void;
} {
  const [fullPage, setFullPage] = useState(readStoredWorkOrderFullPage);

  const toggleFullPage = useCallback(() => {
    setFullPage((current) => {
      const next = !current;
      persistWorkOrderFullPage(next);
      return next;
    });
  }, []);

  return { fullPage, toggleFullPage };
}
