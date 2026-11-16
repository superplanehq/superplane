import { useId, useState } from "react";

export interface ColumnAutomationDisclosureState {
  open: boolean;
  setOpen: (open: boolean) => void;
  contentId: string;
}

function readPreference(key: string): boolean | null {
  try {
    const value = window.localStorage.getItem(key);
    return value === "expanded" ? true : value === "collapsed" ? false : null;
  } catch {
    return null;
  }
}

export function useColumnAutomationCollapsePreference(
  identity: string,
  automationCount: number,
): ColumnAutomationDisclosureState {
  const key = `factories-column-automation-disclosure:${identity}`;
  const contentId = useId();
  const [preferences, setPreferences] = useState<Record<string, boolean | null>>({});
  if (!(key in preferences)) {
    setPreferences({ ...preferences, [key]: readPreference(key) });
  }
  const open = preferences[key] ?? automationCount < 3;
  const setOpen = (next: boolean) => {
    setPreferences((current) => ({ ...current, [key]: next }));
    try {
      window.localStorage.setItem(key, next ? "expanded" : "collapsed");
    } catch {
      // Keep the explicit choice in this session when storage is unavailable.
    }
  };
  return { open, setOpen, contentId };
}
