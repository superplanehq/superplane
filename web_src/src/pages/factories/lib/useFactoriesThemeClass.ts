import { useEffect } from "react";

const THEME_CLASS = "theme-factories";

/**
 * Applies the `theme-factories` class to `document.body` while the calling
 * component is mounted. Any element rendered via a portal (dialogs, dropdowns,
 * toasts) is a child of body, so the scoped design tokens defined in App.css
 * flow through automatically.
 *
 * A shared ref-count guards against races when multiple factories subtrees
 * mount concurrently (e.g. during route transitions), so the class only comes
 * off once the last consumer has unmounted.
 */
let mountCount = 0;

export function useFactoriesThemeClass() {
  useEffect(() => {
    mountCount += 1;
    document.body.classList.add(THEME_CLASS);
    return () => {
      mountCount -= 1;
      if (mountCount <= 0) {
        mountCount = 0;
        document.body.classList.remove(THEME_CLASS);
      }
    };
  }, []);
}
