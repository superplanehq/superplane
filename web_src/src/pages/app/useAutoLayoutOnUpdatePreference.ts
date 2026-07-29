import { useCallback, useState } from "react";
import { readStoredBoolean } from "./viewState";

const CANVAS_AUTO_LAYOUT_ON_UPDATE_STORAGE_KEY = "canvas-auto-layout-on-update-enabled";

export function useAutoLayoutOnUpdatePreference() {
  const [isAutoLayoutOnUpdateEnabled, setIsAutoLayoutOnUpdateEnabled] = useState(() =>
    readStoredBoolean(CANVAS_AUTO_LAYOUT_ON_UPDATE_STORAGE_KEY),
  );

  const handleToggleAutoLayoutOnUpdate = useCallback(() => {
    const newValue = !isAutoLayoutOnUpdateEnabled;
    setIsAutoLayoutOnUpdateEnabled(newValue);
    if (typeof window !== "undefined") {
      window.localStorage.setItem(CANVAS_AUTO_LAYOUT_ON_UPDATE_STORAGE_KEY, JSON.stringify(newValue));
    }
  }, [isAutoLayoutOnUpdateEnabled]);

  return { handleToggleAutoLayoutOnUpdate, isAutoLayoutOnUpdateEnabled };
}
