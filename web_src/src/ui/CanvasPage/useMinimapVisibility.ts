import { useEffect, useState } from "react";

const CANVAS_MINIMAP_VISIBLE_STORAGE_KEY = "canvasMinimapVisible";

export function useMinimapVisibility(defaultValue = false) {
  const [isMinimapVisible, setIsMinimapVisible] = useState(() => {
    if (typeof window === "undefined") {
      return defaultValue;
    }

    const storedMinimapState = window.localStorage.getItem(CANVAS_MINIMAP_VISIBLE_STORAGE_KEY);
    if (storedMinimapState === null) {
      return defaultValue;
    }

    try {
      return JSON.parse(storedMinimapState);
    } catch (error) {
      console.warn("Failed to parse minimap visibility state:", error);
      return defaultValue;
    }
  });

  useEffect(() => {
    if (typeof window === "undefined") {
      return;
    }

    window.localStorage.setItem(CANVAS_MINIMAP_VISIBLE_STORAGE_KEY, JSON.stringify(isMinimapVisible));
  }, [isMinimapVisible]);

  return { isMinimapVisible, setIsMinimapVisible };
}
