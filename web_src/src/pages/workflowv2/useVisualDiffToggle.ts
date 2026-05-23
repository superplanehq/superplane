import { useCallback, useState } from "react";

const STORAGE_KEY = "visual-diff-enabled";

export function useVisualDiffToggle() {
  const [enabled, setEnabled] = useState(() => {
    const stored = localStorage.getItem(STORAGE_KEY);
    return stored === null ? true : stored === "true";
  });

  const toggle = useCallback(() => {
    setEnabled((prev) => {
      const next = !prev;
      localStorage.setItem(STORAGE_KEY, String(next));
      return next;
    });
  }, []);

  return { visualDiffEnabled: enabled, toggleVisualDiff: toggle };
}
