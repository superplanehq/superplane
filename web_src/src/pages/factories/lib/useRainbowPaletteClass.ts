import { useEffect } from "react";

const PALETTE_CLASS = "theme-factories-palette-rainbow";

/**
 * Applies `theme-factories-palette-rainbow` to `document.body` while the
 * calling component is mounted. Overview is the only caller; navigation and
 * portal content pick up the tokens until the route unmounts.
 *
 * A shared ref-count guards against races when multiple overview subtrees
 * mount concurrently (e.g. during route transitions), so the class only comes
 * off once the last consumer has unmounted.
 */
let mountCount = 0;

export function useRainbowPaletteClass() {
  useEffect(() => {
    mountCount += 1;
    document.body.classList.add(PALETTE_CLASS);
    return () => {
      mountCount -= 1;
      if (mountCount <= 0) {
        mountCount = 0;
        document.body.classList.remove(PALETTE_CLASS);
      }
    };
  }, []);
}
