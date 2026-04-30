import { useMemo } from "react";

export function useApplyShortcutHint(): string {
  return useMemo(() => {
    if (typeof navigator === "undefined") {
      return "Ctrl+Enter";
    }

    const isMacPlatform = /Mac|iPhone|iPad|iPod/i.test(`${navigator.platform} ${navigator.userAgent}`);
    return isMacPlatform ? "Cmd+Enter" : "Ctrl+Enter";
  }, []);
}
