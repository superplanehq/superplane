import { useEffect, useState } from "react";

export const AGENT_SIDEBAR_WIDTH_STORAGE_KEY = "agentSidebarWidth";

const DEFAULT_SIDEBAR_WIDTH_PX = 400;

function readInitialSidebarWidthPx(): number {
  if (typeof window === "undefined") {
    return DEFAULT_SIDEBAR_WIDTH_PX;
  }

  const saved = window.localStorage.getItem(AGENT_SIDEBAR_WIDTH_STORAGE_KEY);
  return saved ? parseInt(saved, 10) : DEFAULT_SIDEBAR_WIDTH_PX;
}

export function useSidebarWidth() {
  const [sidebarWidth, setSidebarWidth] = useState(readInitialSidebarWidthPx);

  useEffect(() => {
    window.localStorage.setItem(AGENT_SIDEBAR_WIDTH_STORAGE_KEY, String(sidebarWidth));
  }, [sidebarWidth]);

  return [sidebarWidth, setSidebarWidth] as const;
}
