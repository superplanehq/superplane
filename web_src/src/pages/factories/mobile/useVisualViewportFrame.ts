import { useEffect, useState } from "react";

/**
 * Visible screen inside the layout viewport. Phone shells pin to this frame
 * so the Android address bar cannot scroll the bottom bar with the page.
 */
export type VisualViewportFrame = {
  top: number;
  height: number;
};

const APP_CONTENT_REGION_SELECTOR = "[data-app-content-region]";

export function readVisualViewportFrame(): VisualViewportFrame {
  const viewport = window.visualViewport;
  if (viewport && Number.isFinite(viewport.height) && viewport.height > 0) {
    return {
      top: finiteOrZero(viewport.offsetTop),
      height: viewport.height,
    };
  }
  return { top: 0, height: dynamicViewportHeight() };
}

/**
 * Shift a visual-viewport frame below a banner that is still on screen.
 * A missing or scrolled-away banner leaves the frame unchanged.
 */
export function phoneShellFrame(viewport: VisualViewportFrame, bannerInset: number): VisualViewportFrame {
  const inset = Math.max(0, finiteOrZero(bannerInset));
  return {
    top: viewport.top + inset,
    height: Math.max(0, viewport.height - inset),
  };
}

export function readVisibleBannerInset(): number {
  const region = document.querySelector(APP_CONTENT_REGION_SELECTOR);
  if (!(region instanceof HTMLElement)) {
    return 0;
  }
  return Math.max(0, region.getBoundingClientRect().top);
}

export function useVisualViewportFrame(): VisualViewportFrame {
  const [frame, setFrame] = useState(readVisualViewportFrame);

  useEffect(() => {
    const update = () => {
      setFrame(readVisualViewportFrame());
    };
    update();
    return subscribeToViewportChanges(update);
  }, []);

  return frame;
}

/**
 * Visual viewport frame, moved down when an app banner sits above the route.
 * The shell then fills the visible screen without covering the banner.
 */
export function usePinnedPhoneShellFrame(): VisualViewportFrame {
  const viewport = useVisualViewportFrame();
  const [bannerInset, setBannerInset] = useState(readVisibleBannerInset);

  useEffect(() => {
    const update = () => {
      setBannerInset(readVisibleBannerInset());
    };
    update();
    const region = document.querySelector(APP_CONTENT_REGION_SELECTOR);
    const observer = region && typeof ResizeObserver !== "undefined" ? new ResizeObserver(update) : null;
    if (region) {
      observer?.observe(region);
    }
    const unsubscribe = subscribeToViewportChanges(update);
    return () => {
      observer?.disconnect();
      unsubscribe();
    };
  }, []);

  return phoneShellFrame(viewport, bannerInset);
}

function subscribeToViewportChanges(update: () => void): () => void {
  const viewport = window.visualViewport;
  viewport?.addEventListener("resize", update);
  viewport?.addEventListener("scroll", update);
  window.addEventListener("resize", update);
  return () => {
    viewport?.removeEventListener("resize", update);
    viewport?.removeEventListener("scroll", update);
    window.removeEventListener("resize", update);
  };
}

function dynamicViewportHeight(): number {
  const probe = document.createElement("div");
  probe.style.position = "absolute";
  probe.style.visibility = "hidden";
  probe.style.pointerEvents = "none";
  probe.style.height = "100dvh";
  document.documentElement.appendChild(probe);
  const measured = probe.offsetHeight;
  probe.remove();
  if (measured > 0) {
    return measured;
  }
  return finiteOrZero(window.innerHeight);
}

function finiteOrZero(value: number): number {
  return Number.isFinite(value) ? value : 0;
}
