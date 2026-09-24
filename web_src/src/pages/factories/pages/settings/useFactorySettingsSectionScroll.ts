import { useEffect } from "react";
import { useLocation, useSearchParams } from "react-router";

const SECTION_QUERY = "section";
const MAX_SCROLL_FRAMES = 45;

/**
 * After Find settings navigates with `?section=<id>`, scroll that element into
 * view inside the settings main pane. Retries briefly so async catalog cards
 * (integrations) can mount first.
 */
export function useFactorySettingsSectionScroll() {
  const { pathname } = useLocation();
  const [searchParams] = useSearchParams();
  const sectionId = searchParams.get(SECTION_QUERY);

  useEffect(() => {
    if (!sectionId) {
      return;
    }

    let frames = 0;
    let raf = 0;
    let cancelled = false;

    const tryScroll = () => {
      if (cancelled) {
        return;
      }

      const target = document.getElementById(sectionId);
      if (target) {
        target.scrollIntoView({ block: "start", behavior: "smooth" });
        return;
      }

      frames += 1;
      if (frames < MAX_SCROLL_FRAMES) {
        raf = window.requestAnimationFrame(tryScroll);
      }
    };

    raf = window.requestAnimationFrame(() => {
      raf = window.requestAnimationFrame(tryScroll);
    });

    return () => {
      cancelled = true;
      window.cancelAnimationFrame(raf);
    };
  }, [pathname, sectionId]);
}

export function factorySettingsSectionQuery(anchor: string | undefined): string {
  if (!anchor) {
    return "";
  }
  return `?${SECTION_QUERY}=${encodeURIComponent(anchor)}`;
}
