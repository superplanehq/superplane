import { useCallback, useEffect, useRef, useState } from "react";
import {
  getLastRunNodeDetailTab,
  isRunNodeDetailTabAvailable,
  rememberRunNodeDetailTab,
  resolveRunNodeDetailTab,
  type RunNodeDetailTabAvailability,
  type RunNodeDetailTabKey,
} from "./runNodeDetailModel";

export function useRunNodeDetailTabs(nodeId: string, tabAvailability: RunNodeDetailTabAvailability) {
  const [activeTab, setActiveTab] = useState<RunNodeDetailTabKey>(() => getLastRunNodeDetailTab());
  const previousNodeIdRef = useRef(nodeId);
  const tabSelectionWasFallbackRef = useRef(false);

  const selectTab = useCallback((tab: RunNodeDetailTabKey) => {
    rememberRunNodeDetailTab(tab);
    tabSelectionWasFallbackRef.current = false;
    setActiveTab(tab);
  }, []);

  useEffect(() => {
    const nodeChanged = previousNodeIdRef.current !== nodeId;
    previousNodeIdRef.current = nodeId;

    setActiveTab((current) => {
      const preferred = getLastRunNodeDetailTab();
      const resolved = resolveRunNodeDetailTab(preferred, tabAvailability);
      const currentIsValid = isRunNodeDetailTabAvailable(current, tabAvailability);
      const preferredIsValid = isRunNodeDetailTabAvailable(preferred, tabAvailability);

      if (nodeChanged) {
        tabSelectionWasFallbackRef.current = resolved !== preferred;
        return resolved;
      }

      if (currentIsValid) {
        if (tabSelectionWasFallbackRef.current && preferredIsValid) {
          tabSelectionWasFallbackRef.current = false;
          return preferred;
        }
        return current;
      }

      tabSelectionWasFallbackRef.current = resolved !== preferred;
      return resolved;
    });
  }, [nodeId, tabAvailability]);

  return { activeTab, selectTab };
}

export function useRunNodeDetailEscape(onClose: () => void) {
  useEffect(() => {
    const handleKey = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        onClose();
      }
    };

    window.addEventListener("keydown", handleKey);
    return () => window.removeEventListener("keydown", handleKey);
  }, [onClose]);
}
